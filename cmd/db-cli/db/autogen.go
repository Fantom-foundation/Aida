package db

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

const patchesJsonName = "patches.json"

// AutoGenCommand generates aida-db patches and handles second opera for event generation
var AutoGenCommand = cli.Command{
	Action: autoGen,
	Name:   "autogen",
	Usage:  "autogen generates aida-db periodically",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.DbFlag,
		&utils.CompactDbFlag,
		&utils.GenesisFlag,
		&utils.DbTmpFlag,
		&utils.ChainIDFlag,
		&utils.CacheFlag,
		&utils.OperaDatadirFlag,
		&utils.OutputFlag,
		&logger.LogLevelFlag,
	},
	Description: `
Autogen generates aida-db patches and handles second opera for event generation. Generates event file, which is supplied into generate to create aida-db patch.
`,
}

// autoGen command is used to record/update aida-db periodically
func autoGen(ctx *cli.Context) error {
	var err error
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	log := logger.NewLogger(cfg.LogLevel, "autogen")

	log.Info("Starting Automatic generation")

	// preparing config and directories
	aidaDbTmp, err := prepare(cfg)
	if err != nil {
		return err
	}
	defer func(log *logging.Logger) {
		err = os.RemoveAll(aidaDbTmp)
		if err != nil {
			log.Criticalf("can't remove temporary folder: %v; %v", aidaDbTmp, err)
		}
	}(log)

	// loading epoch range for generation
	var firstEpoch, lastEpoch string
	var newDataReady bool
	firstEpoch, lastEpoch, newDataReady, err = loadGenerationRange(cfg, log)
	if err != nil {
		return err
	}
	if !newDataReady {
		log.Infof("No new data for generation. Source epoch %v (%v), Last generation %v (%v)", firstEpoch, cfg.OperaDatadir, lastEpoch, cfg.Db)
		return nil
	}
	log.Infof("Found new epochs for generation %v - %v", firstEpoch, lastEpoch)

	// stop opera to be able to export events
	err = stopOpera(log)
	if err != nil {
		return err
	}
	log.Info("Generating events")

	cfg.Events, err = generateEvents(cfg, aidaDbTmp, firstEpoch, lastEpoch, log)
	if err != nil {
		return err
	}

	// start opera to load new blocks in parallel
	err = startOpera(log)
	if err != nil {
		return err
	}

	// update target aida-db
	err = Generate(cfg, log)
	if err != nil {
		return err
	}

	// if patch output dir is selected inserting just the patch into there
	if cfg.Output != "" {
		patchPath, err := createPatch(cfg, aidaDbTmp, firstEpoch, lastEpoch)
		if err != nil {
			return err
		}
		log.Infof("Successfully generated patch at: %v", patchPath)
	}

	return nil
}

// createPatch create patch from newly generated data
func createPatch(cfg *utils.Config, aidaDbTmp string, firstEpoch string, lastEpoch string) (string, error) {
	// create a parents of output directory
	err := os.MkdirAll(cfg.Output, 0700)
	if err != nil {
		return "", fmt.Errorf("failed to create %s directory; %s", cfg.DbTmp, err)
	}

	// loadingSourceDBPaths because cfg values are already rewritten to aida-db
	// these databases contain just the patch data
	loadSourceDBPaths(cfg, aidaDbTmp)

	// creating patch
	patchName := "aida-db-" + firstEpoch + "-" + lastEpoch
	cfg.AidaDb = filepath.Join(cfg.Output, patchName)
	// merge UpdateDb into AidaDb
	err = Merge(cfg, []string{cfg.SubstateDb, cfg.UpdateDb, cfg.DeletionDb})
	if err != nil {
		return "", fmt.Errorf("unable to merge into patch; %v", err)
	}

	err = updatePatchesJson(cfg.Output, patchName, firstEpoch)
	if err != nil {
		return "", err
	}

	return cfg.AidaDb, nil
}

// updatePatchesJson updates available patches in a JSON file
func updatePatchesJson(patchDir, patchName, patchEpoch string) error {
	// Check if the JSON file exists
	jsonFilePath := filepath.Join(patchDir, patchesJsonName)
	var patchesJson map[string]string

	// Load previous JSON
	file, err := os.Open(jsonFilePath)
	if err == nil {
		// Unmarshal the JSON
		var fileContent []byte
		fileContent, err = io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("unable to read file %s; %v", patchesJsonName, err)
		}
		err = json.Unmarshal(fileContent, &patchesJson)
		if err != nil {
			return fmt.Errorf("unable to unmarshal json from file %s; %v", patchesJsonName, err)
		}
		// close file
		err = file.Close()
		if err != nil {
			return fmt.Errorf("unable to close %s; %v", patchesJsonName, err)
		}
	}

	// open file for write and delete previous
	file, err = os.OpenFile(jsonFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("unable to open %s; %v", patchesJsonName, err)
	}

	// Initialize the map if it doesn't exist
	if patchesJson == nil {
		patchesJson = make(map[string]string)
	}

	// Add the patchName to the corresponding epoch's array
	patchesJson[patchEpoch] = patchName

	// Convert the map to JSON bytes
	jsonBytes, err := json.Marshal(patchesJson)
	if err != nil {
		return fmt.Errorf("unable to marshal %v; %v", patchesJson, err)
	}

	// Write result
	w := bufio.NewWriter(file)
	_, err = w.Write(jsonBytes)
	if err != nil {
		return fmt.Errorf("unable to write %v; %v", patchesJson, err)
	}
	err = w.Flush()
	if err != nil {
		return fmt.Errorf("unable to flush; %v", err)
	}
	return nil
}

// generateEvents generates events between First and Last epoch numbers from config
func generateEvents(cfg *utils.Config, aidaDbTmp string, firstEpoch string, lastEpoch string, log *logging.Logger) (string, error) {
	eventsFile := "events-" + firstEpoch + "-" + lastEpoch
	eventsPath := filepath.Join(aidaDbTmp, eventsFile)
	log.Debugf("Generating events from %v to %v into %v", firstEpoch, lastEpoch, eventsPath)
	cmd := exec.Command("opera", "--datadir", cfg.OperaDatadir, "export", "events", eventsPath, firstEpoch, lastEpoch)
	err := runCommand(cmd, nil, log)
	if err != nil {
		return "", fmt.Errorf("retrieve last opera epoch trough ipc; %v", err.Error())
	}
	return eventsPath, nil
}

// loadGenerationRange retrieves epoch of last generation and most recent available epoch
func loadGenerationRange(cfg *utils.Config, log *logging.Logger) (string, string, bool, error) {
	var previousEpoch uint64 = 1
	_, err := os.Stat(cfg.Db)
	if !os.IsNotExist(err) {
		// opera was already used for generation starting from the next epoch
		_, previousEpoch, err = GetOperaBlock(cfg)
		if err != nil {
			return "", "", false, fmt.Errorf("unable to retrieve epoch of generation opera in path %v; %v", cfg.Db, err)
		}
		log.Debugf("Generation will start from: %v", previousEpoch)
	}

	nextEpoch, err := getLastOperaEpoch(cfg, log)
	if err != nil {
		return "", "", false, fmt.Errorf("unable to retrieve epoch of source opera in path %v; %v", cfg.OperaDatadir, err)
	}
	// ending generation one epoch sooner to make sure epoch is sealed
	nextEpoch -= 1
	log.Debugf("Last available sealed epoch is %v", nextEpoch)

	// recording of events will start with the following epoch of last recording
	firstEpoch := strconv.FormatUint(previousEpoch, 10)

	// recording of events will stop with last sealed opera
	lastEpoch := strconv.FormatUint(nextEpoch, 10)

	if previousEpoch > nextEpoch {
		return firstEpoch, lastEpoch, false, nil
	}
	return firstEpoch, lastEpoch, true, nil
}

// getLastOperaEpoch loads last epoch from opera
func getLastOperaEpoch(cfg *utils.Config, log *logging.Logger) (uint64, error) {
	var response = ""
	var wg sync.WaitGroup
	resultChan := make(chan string, 10)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case s, ok := <-resultChan:
				if !ok {
					return
				}
				response += s
			}
		}
	}()
	cmd := exec.Command("bash", "-c", "echo '{\"method\": \"eth_getBlockByNumber\", \"params\": [\"latest\", false], \"id\": 1, \"jsonrpc\": \"2.0\"}' | nc -q 0 -U \""+cfg.OperaDatadir+"/opera.ipc\"")
	err := runCommand(cmd, resultChan, log)
	if err != nil {
		return 0, fmt.Errorf("retrieve last opera epoch trough ipc; %v", err.Error())
	}

	// wait until reading of result finishes
	wg.Wait()

	// parse result into json
	var responseJson = make(map[string]interface{})
	err = json.Unmarshal([]byte(response), &responseJson)
	if err != nil {
		return 0, fmt.Errorf("unable to json from %v; %v", response, err.Error())
	}
	result, ok := responseJson["result"]
	if !ok {
		return 0, fmt.Errorf("unable to parse result from %v; %v", responseJson, err.Error())
	}

	epochHex, ok := result.(map[string]interface{})["epoch"]
	if !ok {
		return 0, fmt.Errorf("unable to parse epoch from %v; %v", responseJson, err.Error())
	}

	epochHexCleaned := strings.Replace(epochHex.(string), "0x", "", -1)
	epoch, err := strconv.ParseUint(epochHexCleaned, 16, 64)
	if err != nil {
		return 0, err
	}
	return epoch, nil
}

// startOpera start opera node
func startOpera(log *logging.Logger) error {
	cmd := exec.Command("systemctl", "--user", "start", "opera")
	err := runCommand(cmd, nil, log)
	if err != nil {
		return fmt.Errorf("unable start opera; %v", err.Error())
	}
	return nil
}

// stopOpera stop opera node
func stopOpera(log *logging.Logger) error {
	cmd := exec.Command("systemctl", "--user", "stop", "opera")
	err := runCommand(cmd, nil, log)
	if err != nil {
		return fmt.Errorf("unable stop opera; %v", err.Error())
	}
	return nil
}

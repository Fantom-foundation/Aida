package db

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/Fantom-foundation/Aida/utils"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

const (
	startOperaCommand = "systemctl start opera"
	stopOperaCommand  = "systemctl stop opera"
)

// AutoGenCommand generates aida-db patches and handles second opera for event generation
var AutoGenCommand = cli.Command{
	Action: autoGen,
	Name:   "autogen",
	Usage:  "autogen generates aida-db periodically",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.DbFlag,
		&utils.GenesisFlag,
		&utils.DbTmpFlag,
		&utils.ChainIDFlag,
		&utils.CacheFlag,
		&utils.OperaDatadirFlag,
		&utils.OutputFlag,
		&utils.LogLevelFlag,
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

	log := utils.NewLogger(cfg.LogLevel, "autogen")

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
	firstEpoch, lastEpoch, err = loadGenerationRange(cfg, log)

	err = stopOpera(log)
	if err != nil {
		return err
	}

	cfg.Events, err = generateEvents(cfg, aidaDbTmp, firstEpoch, lastEpoch, log)

	// update target aida-db
	err = Generate(cfg, log)
	if err != nil {
		return err
	}

	err = startOpera(log)
	if err != nil {
		return err
	}

	// if patch output dir is selected inserting just the patch into there
	if cfg.Output != "" {
		err = createPatch(cfg, aidaDbTmp, firstEpoch, lastEpoch)
		if err != nil {
			return err
		}
	}

	return nil
}

// createPatch create patch from newly generated data
func createPatch(cfg *utils.Config, aidaDbTmp string, firstEpoch string, lastEpoch string) error {
	// create a parents of output directory
	err := os.MkdirAll(cfg.Output, 0700)
	if err != nil {
		return fmt.Errorf("failed to create %s directory; %s", cfg.DbTmp, err)
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
		return err
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
func loadGenerationRange(cfg *utils.Config, log *logging.Logger) (string, string, error) {
	var previousEpoch uint64 = 1
	_, err := os.Stat(cfg.Db)
	if !os.IsNotExist(err) {
		// opera was already used for generation starting from the next epoch
		_, previousEpoch, err = GetOperaBlock(cfg)
		if err != nil {
			return "", "", fmt.Errorf("unable to retrieve epoch of generation opera in path %v; %v", cfg.Db, err)
		}
		log.Debugf("Previous generation ended with epoch: %v", previousEpoch)
		previousEpoch += 1
		log.Debugf("Generation will start from: %v", previousEpoch)
	}

	nextEpoch, err := getLastOperaEpoch(cfg, log)
	if err != nil {
		return "", "", fmt.Errorf("unable to retrieve epoch of source opera in path %v; %v", cfg.OperaDatadir, err)
	}
	// ending generation one epoch sooner to make sure epoch is sealed
	nextEpoch -= 1
	log.Debugf("Last available sealed epoch is %v", nextEpoch)

	if previousEpoch > nextEpoch {
		return "", "", fmt.Errorf("source epoch %v (%v) can't be lower than epoch of last generation %v (%v)", nextEpoch, cfg.OperaDatadir, previousEpoch, cfg.Db)
	}

	// recording of events will start with the following epoch of last recording
	firstEpoch := strconv.FormatUint(previousEpoch, 10)

	// recording of events will stop with last sealed opera
	lastEpoch := strconv.FormatUint(nextEpoch, 10)
	return firstEpoch, lastEpoch, nil
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
	cmd := exec.Command("echo '{\n    \"method\": \"eth_getBlockByNumber\",\n    \"params\": [\"latest\", false],\n    \"id\": 1,\n    \"jsonrpc\": \"2.0\"}' | nc -q 0 -U \"" + cfg.OperaDatadir + "/opera.ipc\"")
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
		return 0, fmt.Errorf("unable to json from %v; %v", responseJson, err.Error())
	}

	epochHex, ok := responseJson["epoch"]
	if !ok {
		return 0, fmt.Errorf("unable to parse epoch from %v; %v", responseJson, err.Error())
	}
	epoch, err := strconv.ParseUint(epochHex.(string), 16, 64)
	if err != nil {
		return 0, err
	}
	return epoch, nil
}

// startOpera start opera node
func startOpera(log *logging.Logger) error {
	cmd := exec.Command(startOperaCommand)
	err := runCommand(cmd, nil, log)
	if err != nil {
		return fmt.Errorf("unable start opera; %v", err.Error())
	}
	return nil
}

// stopOpera stop opera node
func stopOpera(log *logging.Logger) error {
	cmd := exec.Command(stopOperaCommand)
	err := runCommand(cmd, nil, log)
	if err != nil {
		return fmt.Errorf("unable stop opera; %v", err.Error())
	}
	return nil
}

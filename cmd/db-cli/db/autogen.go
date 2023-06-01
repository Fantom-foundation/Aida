package db

import (
	"archive/tar"
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

	"github.com/Fantom-foundation/Aida/cmd/db-cli/flags"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/klauspost/compress/gzip"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

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
		&utils.UpdateBufferSizeFlag,
		&utils.ChannelBufferSizeFlag,
		&utils.OperaDatadirFlag,
		&utils.OutputFlag,
		&logger.LogLevelFlag,
		&flags.SkipMetadata,
	},
	Description: `
AutoGen generates aida-db patches and handles second opera for event generation. Generates event file, which is supplied into generate to create aida-db patch.
`,
}

const patchesJsonName = "patches.json"

// autoGen command is used to record/update aida-db periodically
func autoGen(ctx *cli.Context) error {
	var err error
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	log := logger.NewLogger(cfg.LogLevel, "autoGen")

	log.Info("Starting Automatic generation")

	// preparing config and directories
	aidaDbTmp, err := prepareDbDirs(cfg)
	if err != nil {
		return err
	}

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

	var mdi *Metadata
	// update target aida-db
	mdi, err = Generate(cfg, log)
	if err != nil {
		return err
	}

	// if patch output dir is selected inserting patch.tar.gz, patch.tar.gz.md5 into there and updating patches.json
	if cfg.Output != "" {
		mdi.dbType = patchType
		patchTarPath, err := createPatch(cfg, aidaDbTmp, firstEpoch, lastEpoch, cfg.First, cfg.Last, log, mdi)

		if err != nil {
			return err
		}
		log.Infof("Successfully generated patch at: %v", patchTarPath)
	}

	// remove temporary folder only if generation completed successfully
	err = os.RemoveAll(aidaDbTmp)
	if err != nil {
		log.Criticalf("can't remove temporary folder: %v; %v", aidaDbTmp, err)
	}

	return nil
}

// createPatch create patch from newly generated data
func createPatch(cfg *utils.Config, aidaDbTmp string, firstEpoch string, lastEpoch string, firstBlock uint64, lastBlock uint64, log *logging.Logger, mdi *Metadata) (string, error) {
	// create a parents of output directory
	err := os.MkdirAll(cfg.Output, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create %s directory; %s", cfg.DbTmp, err)
	}

	// loadingSourceDBPaths because cfg values are already rewritten to aida-db
	// these databases contain just the patch data
	loadSourceDBPaths(cfg, aidaDbTmp)

	// creating patch name
	// add leading zeroes to filename to make it sortable
	patchName := "aida-db-" + fmt.Sprintf("%09s", lastEpoch)
	patchPath := filepath.Join(cfg.Output, patchName)
	// cfg.AidaDb is now pointing to patch this is needed for Merge function
	cfg.AidaDb = patchPath

	// merge UpdateDb into AidaDb
	err = Merge(cfg, []string{cfg.SubstateDb, cfg.UpdateDb, cfg.DeletionDb}, mdi)
	if err != nil {
		return "", fmt.Errorf("unable to merge into patch; %v", err)
	}

	patchTarName := patchName + ".tar.gz"
	patchTarPath := filepath.Join(cfg.Output, patchTarName)
	err = createPatchTarGz(patchPath, cfg.Output, patchTarName, log)
	if err != nil {
		return "", fmt.Errorf("unable to create patch tar.gz of %s; %v", patchPath, err)
	}

	log.Noticef("Patch %s generated successfully: %d(%s) - %d(%s) ", patchTarName, firstBlock, firstEpoch, lastBlock, lastEpoch)

	err = updatePatchesJson(cfg.Output, patchTarName, firstEpoch, lastEpoch, firstBlock, lastBlock, log)
	if err != nil {
		return "", err
	}

	err = storeMd5sum(patchTarPath, log)
	if err != nil {
		return "", err
	}

	// remove patchFiles
	err = os.RemoveAll(patchPath)
	if err != nil {
		return "", err
	}

	return patchTarPath, nil
}

// storeMd5sum store md5sum of aida-db patch into a file
func storeMd5sum(filePath string, log *logging.Logger) error {
	md5sum, err := calculateMd5sum(filePath, log)
	if err != nil {
		return err
	}

	md5FilePath := filePath + ".md5"

	var file *os.File
	file, err = os.OpenFile(md5FilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("unable to create %s; %v", md5FilePath, err)
	}

	// Write the result
	w := bufio.NewWriter(file)
	_, err = w.Write([]byte(md5sum))
	if err != nil {
		return fmt.Errorf("unable to write %s into %s; %v", md5sum, md5FilePath, err)
	}
	err = w.Flush()
	if err != nil {
		return fmt.Errorf("unable to flush %s; %v", md5FilePath, err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("unable to close %s; %v", patchesJsonName, err)
	}

	return nil
}

// calculateMd5sum calculates md5sum of a file
func calculateMd5sum(filePath string, log *logging.Logger) (string, error) {
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

	cmd := exec.Command("bash", "-c", "md5sum "+filePath)
	err := runCommand(cmd, resultChan, log)
	if err != nil {
		return "", fmt.Errorf("unable sum md5; %v", err.Error())
	}

	// wait until reading of result finishes
	wg.Wait()

	md5 := getFirstWord(response)
	if md5 == "" {
		return "", fmt.Errorf("unable to calculate md5sum")
	}

	// md5 is always 32 characters long
	if len(md5) != 32 {
		return "", fmt.Errorf("unable to generate correct md5sum; Error: %v is not md5", md5)
	}

	return md5, nil
}

// createPatchTarGz compresses patch file into tar.gz
func createPatchTarGz(patchPath string, patchParentPath string, patchTarName string, log *logging.Logger) error {
	log.Noticef("Generating compressed %v", patchTarName)
	err := createTarGz(patchPath, patchParentPath, patchTarName)
	if err != nil {
		return fmt.Errorf("unable to compress %v; %v", patchTarName, err)
	}
	return nil
}

// updatePatchesJson update patches.json file
func updatePatchesJson(patchDir, patchName, fromEpoch string, toEpoch string, fromBlock uint64, toBlock uint64, log *logging.Logger) error {
	jsonFilePath := filepath.Join(patchDir, patchesJsonName)
	var patchesJson []map[string]string

	// Attempt to load previous JSON
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
		// Close the file
		err = file.Close()
		if err != nil {
			return fmt.Errorf("unable to close %s; %v", patchesJsonName, err)
		}
	}

	// Open file for write and delete previous contents
	file, err = os.OpenFile(jsonFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("unable to open %s; %v", patchesJsonName, err)
	}

	// Initialize the array if it doesn't exist
	if patchesJson == nil {
		patchesJson = make([]map[string]string, 0)
	}

	// Create a new patch object
	newPatch := map[string]string{
		"fileName":  patchName,
		"fromBlock": strconv.FormatUint(fromBlock, 10),
		"toBlock":   strconv.FormatUint(toBlock, 10),
		"fromEpoch": fromEpoch,
		"toEpoch":   toEpoch,
	}

	// Append the new patch to the array
	patchesJson = append(patchesJson, newPatch)

	// Convert the array to JSON bytes
	jsonBytes, err := json.Marshal(patchesJson)
	if err != nil {
		return fmt.Errorf("unable to marshal %v; %v", patchesJson, err)
	}

	// Write the result
	w := bufio.NewWriter(file)
	_, err = w.Write(jsonBytes)
	if err != nil {
		return fmt.Errorf("unable to write %v into %s; %v", patchesJson, jsonFilePath, err)
	}
	err = w.Flush()
	if err != nil {
		return fmt.Errorf("unable to flush %v; %v", patchesJson, err)
	}

	// Close the file
	err = file.Close()
	if err != nil {
		return fmt.Errorf("unable to close %s; %v", patchesJsonName, err)
	}

	log.Noticef("Updated %s in %s with new patch:\n%v\n", patchesJsonName, jsonFilePath, newPatch)

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
		// !!! returning number one block greater than actual block
		_, previousEpoch, err = GetOperaBlockAndEpoch(cfg)
		if err != nil {
			return "", "", false, fmt.Errorf("unable to retrieve epoch of generation opera in path %v; %v", cfg.Db, err)
		}
		log.Debugf("Generation will start from: %v", previousEpoch)
	}

	nextEpoch, err := getLastEpochFromRunningOpera(cfg, log)
	if err != nil {
		return "", "", false, fmt.Errorf("unable to retrieve epoch of source opera in path %v; %v", cfg.OperaDatadir, err)
	}
	// ending generation one epoch sooner to make sure epoch is sealed
	nextEpoch -= 1
	log.Debugf("Last available sealed epoch is %v", nextEpoch)

	// recording of events will stop with last sealed opera
	lastEpoch := strconv.FormatUint(nextEpoch, 10)

	if previousEpoch > nextEpoch {
		// since getBlockAndEpoch returns off by one epoch number label
		// needs to be fixed in no need epochs are available
		firstEpoch := strconv.FormatUint(previousEpoch-1, 10)
		return firstEpoch, lastEpoch, false, nil
	}

	// recording of events will start with the following epoch of last recording
	firstEpoch := strconv.FormatUint(previousEpoch, 10)

	return firstEpoch, lastEpoch, true, nil
}

// getLastEpochFromRunningOpera loads last epoch from running opera
func getLastEpochFromRunningOpera(cfg *utils.Config, log *logging.Logger) (uint64, error) {
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

// getFirstWord retrieves first word from string
func getFirstWord(str string) string {
	words := strings.Fields(str)
	if len(words) > 0 {
		return words[0]
	}
	return ""
}

// createTarGz create tar gz of given file/folder
func createTarGz(dirPath, outputPath, outputName string) error {
	// create a parents of temporary directory
	err := os.MkdirAll(outputPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create %s directory; %s", outputPath, err)
	}

	// Create the output file
	file, err := os.Create(filepath.Join(outputPath, outputName))
	if err != nil {
		return err
	}
	defer file.Close()

	// Create the gzip writer
	gw := gzip.NewWriter(file)
	defer gw.Close()

	// Create the tar writer
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Get the base name of the directory
	dirName := filepath.Base(dirPath)

	// Walk through the directory recursively
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a new tar header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Update the header's name to include the directory
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}
		header.Name = filepath.Join(dirName, relPath)

		// Write the header
		err = tw.WriteHeader(header)
		if err != nil {
			return err
		}

		// If it's not a directory, write the file content
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			// Copy the file content to the tar writer
			_, err = io.Copy(tw, file)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

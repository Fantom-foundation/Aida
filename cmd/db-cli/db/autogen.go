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

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/klauspost/compress/gzip"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

const patchesJsonName = "patches.json"

// AutoGenCommand generates aida-db patches and handles second opera for event generation
var AutoGenCommand = cli.Command{
	Action: autogen,
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
	},
	Description: `
AutoGen generates aida-db patches and handles second opera for event generation. Generates event file, which is supplied into generate to create aida-db patch.
`,
}

type automator struct {
	cfg                   *utils.Config
	log                   *logging.Logger
	generator             *generator
	firstEpoch, lastEpoch uint64
}

// autogen command is used to record/update aida-db periodically
func autogen(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	cfg.Workers = substate.WorkersFlag.Value

	log := logger.NewLogger(cfg.LogLevel, "autoGen")

	// preparing config and directories
	aidaDbTmp, err := prepareDbDirs(cfg)
	if err != nil {
		return err
	}

	a := &automator{
		cfg:       cfg,
		log:       log,
		generator: newGenerator(ctx, cfg, aidaDbTmp),
	}

	return a.generate()

}

// generate AidaDb
func (a *automator) generate() error {
	var (
		newDataReady bool
		err          error
	)

	newDataReady, err = a.loadGenerationRange()
	if err != nil {
		return err
	}

	if !newDataReady {
		a.log.Warningf("No new data for generation. Source epoch %v (%v), Last generation %v (%v)", a.firstEpoch, a.cfg.OperaDatadir, a.lastEpoch, a.cfg.Db)
		return nil
	}

	a.log.Noticef("Found new epochs for generation %v - %v", a.firstEpoch, a.lastEpoch)

	// stop opera to be able to export events
	err = stopDaemonOpera(a.log)
	if err != nil {
		return err
	}
	a.log.Notice("Generating events")

	err = a.generator.opera.generateEvents(a.firstEpoch, a.lastEpoch, a.generator.aidaDbTmp)
	if err != nil {
		return err
	}

	// start opera to load new blocks in parallel
	err = startDaemonOpera(a.log)
	if err != nil {
		return err
	}

	err = a.generator.Generate()
	if err != nil {
		return err
	}

	// if patch output dir is selected inserting patch.tar.gz, patch.tar.gz.md5 into there and updating patches.json
	if a.cfg.Output != "" {
		// todo metadata
		patchTarPath, err := a.createPatch()
		if err != nil {
			return err
		}

		a.log.Noticef("Successfully generated patch at: %v", patchTarPath)
	}

	// remove temporary folder only if generation completed successfully
	err = os.RemoveAll(a.generator.aidaDbTmp)
	if err != nil {
		a.log.Criticalf("can't remove temporary folder: %v; %v", a.generator.aidaDbTmp, err)
	}

	return nil
}

// loadGenerationRange retrieves epoch of last generation and most recent available epoch
func (a *automator) loadGenerationRange() (bool, error) {
	a.firstEpoch = 1
	_, err := os.Stat(a.cfg.Db)
	if !os.IsNotExist(err) {
		// opera was already used for generation starting from the next epoch
		// !!! returning number one block greater than actual block
		_, a.firstEpoch, err = GetOperaBlockAndEpoch(a.cfg)
		if err != nil {
			return false, fmt.Errorf("unable to retrieve epoch of generation opera in path %v; %v", a.cfg.Db, err)
		}
		a.log.Debugf("Generation will start from: %v", a.firstEpoch)
	}

	a.lastEpoch, err = a.getLastEpochFromRunningOpera()
	if err != nil {
		return false, fmt.Errorf("unable to retrieve epoch of source opera in path %v; %v", a.cfg.OperaDatadir, err)
	}
	// ending generation one epoch sooner to make sure epoch is sealed
	a.lastEpoch -= 1
	a.log.Debugf("Last available sealed epoch is %v", a.lastEpoch)

	if a.firstEpoch > a.lastEpoch {
		// since getLatestBlockAndEpoch returns off by one epoch number label
		// needs to be fixed in no need epochs are available
		a.firstEpoch = a.firstEpoch - 1
		return false, nil
	}

	return true, nil
}

// getLastEpochFromRunningOpera loads last epoch from running opera
func (a *automator) getLastEpochFromRunningOpera() (uint64, error) {
	var response string
	var wg = new(sync.WaitGroup)
	var resultChan = make(chan string, 10)

	wg.Add(1)
	go a.createResponse(wg, &response, resultChan)

	cmd := exec.Command("bash", "-c", "echo '{\"method\": \"eth_getBlockByNumber\", \"params\": [\"latest\", false], \"id\": 1, \"jsonrpc\": \"2.0\"}' | nc -q 0 -U \""+a.cfg.OperaDatadir+"/opera.ipc\"")
	err := runCommand(cmd, resultChan, a.log)
	if err != nil {
		return 0, fmt.Errorf("retrieve last opera epoch trough ipc; %v", err.Error())
	}

	// wait until reading of result finishes
	wg.Wait()

	// parse result into json
	return a.parseIntoJson(response)
}

// createResponse waits for response from getBlockByNumber cmd and then writes it into string
func (a *automator) createResponse(wg *sync.WaitGroup, response *string, resultChan chan string) {
	defer wg.Done()
	for {
		select {
		case s, ok := <-resultChan:
			if !ok {
				return
			}
			*response += s
		}
	}
}

// parseIntoJson creates json-like object (in this cas map[string]interface{}) which will be marshalled later
func (a *automator) parseIntoJson(response string) (uint64, error) {
	var responseJson = make(map[string]interface{})

	err := json.Unmarshal([]byte(response), &responseJson)
	if err != nil {
		return 0, fmt.Errorf("unable to json from %v; %v", response, err.Error())
	}

	result, ok := responseJson["result"]
	if !ok {
		return 0, fmt.Errorf("unable to parse result from %v", responseJson)
	}

	epochHex, ok := result.(map[string]interface{})["epoch"]
	if !ok {
		return 0, fmt.Errorf("unable to parse epoch from %v; %v", responseJson, err)
	}

	epochHexCleaned := strings.Replace(epochHex.(string), "0x", "", -1)
	epoch, err := strconv.ParseUint(epochHexCleaned, 16, 64)
	if err != nil {
		return 0, err
	}
	return epoch, nil
}

// createPatch for updating data in AidaDb
func (a *automator) createPatch() (string, error) {
	// create a parents of output directory
	err := os.MkdirAll(a.cfg.Output, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create %s directory; %s", a.cfg.DbTmp, err)
	}

	// loadingSourceDBPaths because cfg values are already rewritten to aida-db
	// these databases contain just the patch data
	loadSourceDBPaths(a.cfg, a.generator.aidaDbTmp)

	// creating patch name
	// add leading zeroes to filename to make it sortable
	patchName := fmt.Sprintf("aida-db-%09s", strconv.FormatUint(a.lastEpoch, 10))
	patchPath := filepath.Join(a.cfg.Output, patchName)

	// cfg.AidaDb is now pointing to patch this is needed for Merge function
	a.cfg.AidaDb = patchPath

	// merge UpdateDb into AidaDb
	err = a.mergePatch()
	if err != nil {
		return "", fmt.Errorf("unable to merge into patch; %v", err)
	}

	patchTarName := fmt.Sprintf("%v.tar.gz", patchName)
	patchTarPath := filepath.Join(a.cfg.Output, patchTarName)

	err = a.createPatchTarGz(patchPath, patchTarName)
	if err != nil {
		return "", fmt.Errorf("unable to create patch tar.gz of %s; %v", patchPath, err)
	}

	a.log.Noticef("Patch %s generated successfully: %d(%s) - %d(%s) ", patchTarName, a.cfg.First, a.firstEpoch, a.cfg.Last, a.lastEpoch)

	err = a.updatePatchesJson(patchTarName)
	if err != nil {
		return "", err
	}

	err = a.storeMd5sum(patchTarPath)
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

// mergePatch into existing AidaDb
func (a *automator) mergePatch() error {
	// open targetDb
	targetDb, err := rawdb.NewLevelDBDatabase(a.cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open targetDb; %v", err)
	}

	sourceDbPaths := []string{a.cfg.SubstateDb, a.cfg.UpdateDb, a.cfg.DeletionDb}

	dbs, err := openSourceDatabases(sourceDbPaths)
	if err != nil {
		return err
	}

	m := newMerger(a.cfg, targetDb, dbs, sourceDbPaths)

	return m.merge()
}

// updatePatchesJson with newly acquired patch
func (a *automator) updatePatchesJson(fileName string) error {
	jsonFilePath := filepath.Join(a.cfg.Output, patchesJsonName)
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
		"fileName":  fileName,
		"fromBlock": strconv.FormatUint(a.cfg.First, 10),
		"toBlock":   strconv.FormatUint(a.cfg.Last, 10),
		"fromEpoch": strconv.FormatUint(a.firstEpoch, 10),
		"toEpoch":   strconv.FormatUint(a.lastEpoch, 10),
	}

	// Append the new patch to the array
	patchesJson = append(patchesJson, newPatch)

	if err = a.doUpdatePatchesJson(patchesJson, file); err != nil {
		return err
	}

	a.log.Noticef("Updated %s in %s with new patch:\n%v\n", patchesJsonName, jsonFilePath, newPatch)
	return nil
}

// doUpdatePatchesJson with newly acquired patch
func (a *automator) doUpdatePatchesJson(patchesJson []map[string]string, file *os.File) error {
	// Convert the array to JSON bytes
	jsonBytes, err := json.Marshal(patchesJson)
	if err != nil {
		return fmt.Errorf("unable to marshal %v; %v", patchesJson, err)
	}

	// Write the result
	w := bufio.NewWriter(file)
	_, err = w.Write(jsonBytes)
	if err != nil {
		return fmt.Errorf("unable to write %v; %v", patchesJson, err)
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

	return nil
}

// createPatchTarGz compresses patch file into tar.gz
func (a *automator) createPatchTarGz(filePath string, fileName string) error {
	a.log.Noticef("Generating compressed %v", fileName)
	err := a.createTarGz(filePath, fileName)
	if err != nil {
		return fmt.Errorf("unable to compress %v; %v", fileName, err)
	}
	return nil
}

// storeMd5sum of patch.tar.gz file
func (a *automator) storeMd5sum(filePath string) error {
	md5sum, err := calculateMD5Sum(filePath)
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

// createTarGz create tar gz of given file/folder
func (a *automator) createTarGz(filePath string, fileName string) interface{} {
	// create a parents of temporary directory
	err := os.MkdirAll(a.cfg.Output, 0755)
	if err != nil {
		return fmt.Errorf("failed to create %s directory; %s", a.cfg.Output, err)
	}

	// Create the output file
	file, err := os.Create(filepath.Join(a.cfg.Output, fileName))
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

	// Walk through the directory recursively
	return a.walkFilePath(tw, filePath)
}

// walkFilePath through the directory of patch.tar.gz file recursively
func (a *automator) walkFilePath(tw *tar.Writer, filePath string) error {
	// Get the base name of the directory
	dirName := filepath.Base(filePath)

	return filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a new tar header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Update the header's name to include the directory
		relPath, err := filepath.Rel(filePath, path)
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
}

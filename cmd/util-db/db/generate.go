package db

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Fantom-foundation/Aida/cmd/util-updateset/updateset"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

const (
	// default updateSet interval
	updateSetInterval = 1_000_000
	// number of lines which are kept in memory in case command fails
	commandOutputLimit = 50

	patchesJsonName = "patches.json"
)

// GenerateCommand data structure for the replay app
var GenerateCommand = cli.Command{
	Action: generate,
	Name:   "generate",
	Usage:  "generates aida-db from given events",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&utils.GenesisFlag,
		&utils.WorldStateFlag,
		&utils.OperaBinaryFlag,
		&utils.OutputFlag,
		&utils.TargetEpochFlag,
		&utils.UpdateBufferSizeFlag,
		&substate.WorkersFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The db generate command requires events as an argument:
<events>

<events> are fed into the opera database (either existing or genesis needs to be specified), processing them generates updated aida-db.`,
}

type generator struct {
	cfg         *utils.Config
	log         *logging.Logger
	md          *utils.AidaDbMetadata
	aidaDb      ethdb.Database
	opera       *aidaOpera
	stopAtEpoch uint64
	dbHash      []byte
	start       time.Time
}

// generate AidaDb
func generate(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return fmt.Errorf("cannot create config %v", err)
	}

	g, err := newGenerator(ctx, cfg)
	if err != nil {
		return err
	}

	if err = g.Generate(); err != nil {
		return err
	}

	MustCloseDB(g.aidaDb)

	return printMetadata(g.cfg.AidaDb)
}

// newGenerator returns new instance of generator
func newGenerator(ctx *cli.Context, cfg *utils.Config) (*generator, error) {
	if cfg.AidaDb == "" {
		return nil, fmt.Errorf("you need to specify aida-db (--aida-db)")
	}

	db, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return nil, fmt.Errorf("cannot create new db; %v", err)
	}

	substate.SetSubstateDbBackend(db)

	log := logger.NewLogger(cfg.LogLevel, "AidaDb-Generator")

	return &generator{
		cfg:    cfg,
		log:    log,
		opera:  newAidaOpera(ctx, cfg, log),
		aidaDb: db,
		start:  time.Now(),
	}, nil
}

// Generate is used to record/update aida-db
func (g *generator) Generate() error {
	var err error

	g.log.Noticef("Generation starting for range:  %v (%v) - %v (%v)", g.opera.firstBlock, g.opera.firstEpoch, g.opera.lastBlock, g.opera.lastEpoch)

	deleteDb, updateDb, nextUpdateSetStart, err := g.init()
	if err != nil {
		return err
	}

	if err = g.processDeletedAccounts(deleteDb); err != nil {
		return err
	}

	if err = g.processUpdateSet(deleteDb, updateDb, nextUpdateSetStart); err != nil {
		return err
	}

	start := time.Now()
	g.log.Notice("Starting validation...")

	// after generation is complete, we generateDbHash the db and save it into the patch
	g.dbHash, err = generateDbHash(g.aidaDb, g.cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("cannot generate db hash; %v", err)
	}

	g.log.Noticef("Validation complete. It took: %v", time.Since(start).Round(1*time.Second))
	g.log.Noticef("Total elapsed time: %v", time.Since(g.start).Round(1*time.Second))

	// if patch output dir is selected inserting patch.tar.gz, patch.tar.gz.md5 into there and updating patches.json
	if g.cfg.Output != "" {
		var patchTarPath string
		patchTarPath, err = g.createPatch()
		if err != nil {
			return err
		}

		g.log.Noticef("Successfully generated patch at: %v", patchTarPath)
	}

	g.log.Notice("Generate metadata")
	err = utils.ProcessGenLikeMetadata(g.aidaDb, g.opera.firstBlock, g.opera.lastBlock, g.opera.firstEpoch, g.opera.lastEpoch, g.cfg.ChainID, g.cfg.LogLevel, g.dbHash)
	if err != nil {
		return err
	}

	g.log.Noticef("AidaDb %v generation done", g.cfg.AidaDb)
	g.log.Noticef("Total elapsed time: %v", time.Since(g.start).Round(1*time.Second))

	return nil
}

// init initializes database (DestroyedDb and UpdateDb wrappers) and loads next block for updateset generation
func (g *generator) init() (*substate.DestroyedAccountDB, *substate.UpdateDB, uint64, error) {
	var err error
	deleteDb := substate.NewDestroyedAccountDB(g.aidaDb)

	updateDb := substate.NewUpdateDB(g.aidaDb)

	// set first updateset block
	nextUpdateSetStart := updateDb.GetLastKey()

	if nextUpdateSetStart > 0 {
		g.log.Infof("Previous UpdateSet found - generating from %v", nextUpdateSetStart)
		// generating for next block
		nextUpdateSetStart += 1
	} else {
		g.opera.isNew = true
		g.log.Infof("Previous UpdateSet not found - generating from %v", nextUpdateSetStart)
		_, err = os.Stat(g.cfg.WorldStateDb)
		if os.IsNotExist(err) {
			return nil, nil, 0, fmt.Errorf("you need to specify worldstate extracted before the starting block (--%v)", utils.WorldStateFlag.Name)
		}
	}

	return deleteDb, updateDb, nextUpdateSetStart, err
}

// processDeletedAccounts invokes DeletedAccounts generation and then merges it into AidaDb
func (g *generator) processDeletedAccounts(ddb *substate.DestroyedAccountDB) error {
	var (
		err   error
		start time.Time
	)

	start = time.Now()
	g.log.Noticef("Generating DeletionDb...")

	err = GenDeletedAccountsAction(g.cfg, ddb, 0)
	if err != nil {
		return fmt.Errorf("cannot doGenerations deleted accounts; %v", err)
	}

	g.log.Noticef("Deleted accounts generated successfully. It took: %v", time.Since(start).Round(1*time.Second))
	g.log.Noticef("Total elapsed time: %v", time.Since(g.start).Round(1*time.Second))
	return nil
}

// processUpdateSet invokes UpdateSet generation and then merges it into AidaDb
func (g *generator) processUpdateSet(deleteDb *substate.DestroyedAccountDB, updateDb *substate.UpdateDB, nextUpdateSetStart uint64) error {
	var (
		err   error
		start time.Time
	)

	start = time.Now()
	g.log.Notice("Generating UpdateDb...")

	// nextUpdateSetStart TODO MATEJ probably off by one
	err = updateset.GenUpdateSet(g.cfg, updateDb, deleteDb, nextUpdateSetStart, updateSetInterval)
	if err != nil {
		return fmt.Errorf("cannot doGenerations update-db; %v", err)
	}

	g.log.Noticef("Update-Set generated successfully. It took: %v", time.Since(start).Round(1*time.Second))
	g.log.Noticef("Total elapsed time: %v", time.Since(g.start).Round(1*time.Second))
	return nil

}

// merge sole dbs created in generation into AidaDb
func (g *generator) merge(pathToDb string) error {
	// open sourceDb
	sourceDb, err := rawdb.NewLevelDBDatabase(pathToDb, 1024, 100, "profiling", false)
	if err != nil {
		return err
	}

	m := newMerger(g.cfg, g.aidaDb, []ethdb.Database{sourceDb}, []string{pathToDb}, nil)

	defer func() {
		MustCloseDB(g.aidaDb)
		MustCloseDB(sourceDb)
	}()

	return m.merge()
}

// createPatch for updating data in AidaDb
func (g *generator) createPatch() (string, error) {
	start := time.Now()
	g.log.Notice("Creating patch...")

	// create a parents of output directory
	err := os.MkdirAll(g.cfg.Output, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create %s directory; %s", g.cfg.DbTmp, err)
	}

	// creating patch name
	// add leading zeroes to filename to make it sortable
	patchName := fmt.Sprintf("aida-db-%09s", strconv.FormatUint(g.opera.lastEpoch, 10))
	patchPath := filepath.Join(g.cfg.Output, patchName)

	g.cfg.First = g.opera.firstBlock
	g.cfg.Last = g.opera.lastBlock

	patchDb, err := rawdb.NewLevelDBDatabase(patchPath, 1024, 100, "profiling", false)
	if err != nil {
		return "", fmt.Errorf("cannot open patch db; %v", err)
	}

	err = CreatePatchClone(g.cfg, g.aidaDb, patchDb, g.opera.lastEpoch, g.opera.lastEpoch, g.opera.isNew)

	g.log.Notice("Patch metadata")

	// metadata
	err = utils.ProcessPatchLikeMetadata(patchDb, g.cfg.LogLevel, g.cfg.First, g.cfg.Last, g.opera.firstEpoch,
		g.opera.lastEpoch, g.cfg.ChainID, g.opera.isNew, g.dbHash)
	if err != nil {
		return "", err
	}

	MustCloseDB(patchDb)

	g.log.Noticef("Printing newly generated patch METADATA:")
	err = printMetadata(patchPath)
	if err != nil {
		return "", err
	}

	patchTarName := fmt.Sprintf("%v.tar.gz", patchName)
	patchTarPath := filepath.Join(g.cfg.Output, patchTarName)

	err = g.createPatchTarGz(patchPath, patchTarName)
	if err != nil {
		return "", fmt.Errorf("unable to create patch tar.gz of %s; %v", patchPath, err)
	}

	g.log.Noticef("Patch %s generated successfully: %d(%d) - %d(%d) ", patchTarName, g.cfg.First,
		g.opera.firstEpoch, g.cfg.Last, g.opera.lastEpoch)

	err = g.updatePatchesJson(patchTarName)
	if err != nil {
		return "", err
	}

	err = g.storeMd5sum(patchTarPath)
	if err != nil {
		return "", err
	}

	// remove patchFiles
	err = os.RemoveAll(patchPath)
	if err != nil {
		return "", err
	}

	g.log.Noticef("Patch created successfully. It took: %v", time.Since(start).Round(1*time.Second))
	g.log.Noticef("Total elapsed time: %v", time.Since(g.start).Round(1*time.Second))

	return patchTarPath, nil
}

// updatePatchesJson with newly acquired patch
func (g *generator) updatePatchesJson(fileName string) error {
	jsonFilePath := filepath.Join(g.cfg.Output, patchesJsonName)
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
		"fromBlock": strconv.FormatUint(g.cfg.First, 10),
		"toBlock":   strconv.FormatUint(g.cfg.Last, 10),
		"fromEpoch": strconv.FormatUint(g.opera.firstEpoch, 10),
		"toEpoch":   strconv.FormatUint(g.opera.lastEpoch, 10),
		"nightly":   "true",
	}

	// Append the new patch to the array
	patchesJson = append(patchesJson, newPatch)

	if err = g.doUpdatePatchesJson(patchesJson, file); err != nil {
		return err
	}

	g.log.Noticef("Updated %s in %s with new patch: %v", patchesJsonName, jsonFilePath, newPatch)
	return nil
}

// doUpdatePatchesJson with newly acquired patch
func (g *generator) doUpdatePatchesJson(patchesJson []map[string]string, file *os.File) error {
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
func (g *generator) createPatchTarGz(filePath string, fileName string) error {
	g.log.Noticef("Generating compressed %v", fileName)
	err := g.createTarGz(filePath, fileName)
	if err != nil {
		return fmt.Errorf("unable to compress %v; %v", fileName, err)
	}
	return nil
}

// storeMd5sum of patch.tar.gz file
func (g *generator) storeMd5sum(filePath string) error {
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
func (g *generator) createTarGz(filePath string, fileName string) interface{} {
	// create a parents of temporary directory
	err := os.MkdirAll(g.cfg.Output, 0755)
	if err != nil {
		return fmt.Errorf("failed to create %s directory; %s", g.cfg.Output, err)
	}

	// Create the output file
	file, err := os.Create(filepath.Join(g.cfg.Output, fileName))
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
	return walkFilePath(tw, filePath)
}

// calculatePatchEnd retrieve epoch at which will next patch generation end
func (g *generator) calculatePatchEnd() error {
	err := g.opera.getOperaBlockAndEpoch(false)
	if err != nil {
		return err
	}

	// load last finished epoch to calculate next target
	g.stopAtEpoch = g.opera.lastEpoch

	// next patch will be at least X epochs large
	if g.cfg.ChainID == 250 {
		// mainnet currently takes about 250 epochs per day
		g.stopAtEpoch += 250
	} else {
		// generic value - about 3 days on testnet 4002
		g.stopAtEpoch += 50
	}

	headEpochNumber, err := utils.FindHeadEpochNumber(g.cfg.ChainID)
	if err != nil {
		return err
	}

	// if current generator is too far in history, start generation to the current head
	if headEpochNumber > g.stopAtEpoch {
		g.stopAtEpoch = headEpochNumber
	}

	return nil
}

// walkFilePath through the directory of patch.tar.gz file recursively
func walkFilePath(tw *tar.Writer, filePath string) error {
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

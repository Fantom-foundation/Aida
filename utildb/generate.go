package utildb

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Fantom-foundation/Aida/cmd/util-updateset/updateset"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/urfave/cli/v2"
)

const (
	// default updateSet interval
	updateSetInterval = 1_000_000
	// number of lines which are kept in memory in case command fails
	commandOutputLimit = 50

	patchesJsonName = "patches.json"
)

type Generator struct {
	Cfg          *utils.Config
	ctx          *cli.Context
	Log          logger.Logger
	md           *utils.AidaDbMetadata
	AidaDb       ethdb.Database
	Opera        *aidaOpera
	TargetEpoch  uint64
	dbHash       []byte
	patchTarHash string
	start        time.Time
}

// PrepareManualGenerate prepares generator for manual generation
func (g *Generator) PrepareManualGenerate(ctx *cli.Context, cfg *utils.Config) (err error) {
	if ctx.Args().Len() != 4 {
		return fmt.Errorf("generate command requires exactly 4 arguments - first block, last block, first epoch, last epoch")
	}

	g.Opera.firstBlock, g.Opera.lastBlock, err = utils.SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1), cfg.ChainID)
	if err != nil {
		return err
	}

	g.Opera.FirstEpoch, g.Opera.lastEpoch, err = utils.SetBlockRange(ctx.Args().Get(2), ctx.Args().Get(3), cfg.ChainID)
	if err != nil {
		return err
	}

	if g.Opera.firstBlock > g.Opera.lastBlock {
		return fmt.Errorf("generation range first block %v cannot be greater than available last block %v", g.Opera.firstBlock, g.Opera.lastBlock)
	}

	if g.Opera.FirstEpoch > g.Opera.lastEpoch {
		return fmt.Errorf("generation range first epoch %v cannot be greater than available last epoch %v", g.Opera.FirstEpoch, g.Opera.lastEpoch)
	}

	firstSubstate := substate.NewSubstateDB(g.AidaDb).GetFirstSubstate()
	lastSubstate, err := substate.NewSubstateDB(g.AidaDb).GetLastSubstate()
	if err != nil {
		return fmt.Errorf("cannot get last substate; %v", err)
	}

	if firstSubstate.Env.Number > g.Opera.firstBlock {
		return fmt.Errorf("generation range first block %v cannot be greater than first substate block %v", g.Opera.firstBlock, firstSubstate.Env.Number)
	}

	if lastSubstate.Env.Number < g.Opera.lastBlock {
		return fmt.Errorf("generation range last block %v cannot be greater than last substate block %v", g.Opera.lastBlock, lastSubstate.Env.Number)
	}

	firstAvailableEpoch, err := utils.FindEpochNumber(firstSubstate.Env.Number, g.Cfg.ChainID)
	if err != nil {
		return fmt.Errorf("cannot find first epoch number; %v", err)
	}

	lastAvailableEpoch, err := utils.FindEpochNumber(lastSubstate.Env.Number, g.Cfg.ChainID)
	if err != nil {
		return fmt.Errorf("cannot find last epoch number; %v", err)
	}

	if g.Opera.FirstEpoch < firstAvailableEpoch {
		return fmt.Errorf("generation range first epoch %v cannot be less than first available epoch %v", g.Opera.FirstEpoch, firstAvailableEpoch)
	}

	if g.Opera.lastEpoch > lastAvailableEpoch {
		return fmt.Errorf("generation range last epoch %v cannot be greater than last available epoch %v", g.Opera.lastEpoch, lastAvailableEpoch)
	}
	return nil
}

// NewGenerator returns new instance of generator
func NewGenerator(ctx *cli.Context, cfg *utils.Config) (*Generator, error) {
	if cfg.AidaDb == "" {
		return nil, fmt.Errorf("you need to specify aida-db (--aida-db)")
	}

	db, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return nil, fmt.Errorf("cannot create new db; %v", err)
	}

	substate.SetSubstateDbBackend(db)

	log := logger.NewLogger(cfg.LogLevel, "AidaDb-Generator")

	return &Generator{
		Cfg:    cfg,
		Log:    log,
		Opera:  newAidaOpera(ctx, cfg, log),
		AidaDb: db,
		ctx:    ctx,
		start:  time.Now(),
	}, nil
}

// Generate is used to record/update aida-db
func (g *Generator) Generate() error {
	var err error

	g.Log.Noticef("Generation starting for range:  %v (%v) - %v (%v)", g.Opera.firstBlock, g.Opera.FirstEpoch, g.Opera.lastBlock, g.Opera.lastEpoch)

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

	if g.Cfg.SkipStateHashScrapping {
		g.Log.Noticef("Skipping state hash scraping...")
	} else {
		err = g.runStateHashScraper(g.ctx)
		if err != nil {
			return fmt.Errorf("cannot scrape state hashes; %v", err)
		}
	}

	err = g.runDbHashGeneration(err)
	if err != nil {
		return fmt.Errorf("cannot generate db hash; %v", err)
	}

	g.Log.Notice("Generate metadata for AidaDb...")
	err = utils.ProcessGenLikeMetadata(g.AidaDb, g.Opera.firstBlock, g.Opera.lastBlock, g.Opera.FirstEpoch, g.Opera.lastEpoch, g.Cfg.ChainID, g.Cfg.LogLevel, g.dbHash)
	if err != nil {
		return err
	}

	g.Log.Noticef("AidaDb %v generation done", g.Cfg.AidaDb)

	// if patch output dir is selected inserting patch.tar.gz into there and updating patches.json
	if g.Cfg.Output != "" {
		var patchTarPath string
		patchTarPath, err = g.createPatch()
		if err != nil {
			return err
		}

		g.Log.Noticef("Successfully generated patch at: %v", patchTarPath)
	}
	g.Log.Noticef("Total elapsed time: %v", time.Since(g.start).Round(1*time.Second))

	return nil
}

// runStateHashScraper scrapes state hashes from a node and saves them to a leveldb database
func (g *Generator) runStateHashScraper(ctx *cli.Context) error {
	start := time.Now()
	g.Log.Notice("Starting state-hash scraping...")

	// start opera ipc for state hash scraping
	stopChan := make(chan struct{})
	operaErr := startOperaIpc(g.Cfg, stopChan)

	g.Log.Noticef("Scraping: range %v - %v", g.Opera.firstBlock, g.Opera.lastBlock)

	err := utils.StateHashScraper(ctx.Context, g.Cfg.ChainID, g.Cfg.OperaDb, g.AidaDb, g.Opera.firstBlock, g.Opera.lastBlock, g.Log)
	if err != nil {
		select {
		case oErr, ok := <-operaErr:
			if ok {
				return errors.Join(oErr, err)
			}
			return err
		default:
			return err
		}
	}

	g.Log.Debug("Sending stop signal to opera ipc")
stoppingOperaIpc:
	for {
		select {
		case stopChan <- struct{}{}:
			close(stopChan)
			break stoppingOperaIpc
		case err, ok := <-operaErr:
			if ok {
				return err
			}
		}
	}

	g.Log.Debug("Waiting for opera ipc to finish")
	// wait for child thread to finish
	err, ok := <-operaErr
	if ok {
		return err
	}

	g.Log.Noticef("Hash scraping complete. It took: %v", time.Since(start).Round(1*time.Second))
	g.Log.Noticef("Total elapsed time: %v", time.Since(g.start).Round(1*time.Second))

	return nil
}

// runDbHashGeneration generates db hash of all items in AidaDb
func (g *Generator) runDbHashGeneration(err error) error {
	start := time.Now()
	g.Log.Notice("Starting Db hash generation...")

	// after generation is complete, we generateDbHash the db and save it into the patch
	g.dbHash, err = GenerateDbHash(g.AidaDb, g.Cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("cannot generate db hash; %v", err)
	}

	g.Log.Noticef("Db hash generation complete. It took: %v", time.Since(start).Round(1*time.Second))
	g.Log.Noticef("Total elapsed time: %v", time.Since(g.start).Round(1*time.Second))
	return nil
}

// init initializes database (DestroyedDb and UpdateDb wrappers) and loads next block for updateset generation
func (g *Generator) init() (*substate.DestroyedAccountDB, *substate.UpdateDB, uint64, error) {
	var err error
	deleteDb := substate.NewDestroyedAccountDB(g.AidaDb)

	updateDb := substate.NewUpdateDB(g.AidaDb)

	// set first updateset block
	nextUpdateSetStart, err := updateDb.GetLastKey()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("cannot get last updateset; %v", err)
	}

	if nextUpdateSetStart > 0 {
		g.Log.Infof("Previous UpdateSet found - generating from %v", nextUpdateSetStart)
		// generating for next block
		nextUpdateSetStart += 1
	} else {
		g.Opera.isNew = true
		g.Log.Infof("Previous UpdateSet not found - generating from %v", nextUpdateSetStart)
		_, err = os.Stat(g.Cfg.WorldStateDb)
		if os.IsNotExist(err) {
			return nil, nil, 0, fmt.Errorf("you need to specify worldstate extracted before the starting block (--%v)", utils.WorldStateFlag.Name)
		}
	}

	return deleteDb, updateDb, nextUpdateSetStart, err
}

// processDeletedAccounts invokes DeletedAccounts generation and then merges it into AidaDb
func (g *Generator) processDeletedAccounts(ddb *substate.DestroyedAccountDB) error {
	var (
		err   error
		start time.Time
	)

	start = time.Now()
	g.Log.Noticef("Generating DeletionDb...")

	err = GenDeletedAccountsAction(g.Cfg, ddb, 0, g.Opera.lastBlock)
	if err != nil {
		return fmt.Errorf("cannot doGenerations deleted accounts; %v", err)
	}

	// explicitly release code cache
	state.ReleaseCache()

	g.Log.Noticef("Deleted accounts generated successfully. It took: %v", time.Since(start).Round(1*time.Second))
	g.Log.Noticef("Total elapsed time: %v", time.Since(g.start).Round(1*time.Second))
	return nil
}

// processUpdateSet invokes UpdateSet generation and then merges it into AidaDb
func (g *Generator) processUpdateSet(deleteDb *substate.DestroyedAccountDB, updateDb *substate.UpdateDB, nextUpdateSetStart uint64) error {
	var (
		err   error
		start time.Time
	)

	start = time.Now()
	g.Log.Notice("Generating UpdateDb...")

	err = updateset.GenUpdateSet(g.Cfg, updateDb, deleteDb, nextUpdateSetStart, g.Opera.lastBlock, updateSetInterval)
	if err != nil {
		return fmt.Errorf("cannot doGenerations update-db; %v", err)
	}

	g.Log.Noticef("Update-Set generated successfully. It took: %v", time.Since(start).Round(1*time.Second))
	g.Log.Noticef("Total elapsed time: %v", time.Since(g.start).Round(1*time.Second))
	return nil

}

// merge sole dbs created in generation into AidaDb
func (g *Generator) merge(pathToDb string) error {
	// open sourceDb
	sourceDb, err := rawdb.NewLevelDBDatabase(pathToDb, 1024, 100, "profiling", false)
	if err != nil {
		return err
	}

	m := NewMerger(g.Cfg, g.AidaDb, []ethdb.Database{sourceDb}, []string{pathToDb}, nil)

	defer func() {
		MustCloseDB(g.AidaDb)
		MustCloseDB(sourceDb)
	}()

	return m.Merge()
}

// createPatch for updating data in AidaDb
func (g *Generator) createPatch() (string, error) {
	start := time.Now()
	g.Log.Notice("Creating patch...")

	// create a parents of output directory
	err := os.MkdirAll(g.Cfg.Output, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create %s directory; %s", g.Cfg.DbTmp, err)
	}

	// creating patch name
	// add leading zeroes to filename to make it sortable
	patchName := fmt.Sprintf("%s-%s", strconv.FormatUint(g.Opera.FirstEpoch, 10), strconv.FormatUint(g.Opera.lastEpoch, 10))
	patchPath := filepath.Join(g.Cfg.Output, patchName)

	g.Cfg.TargetDb = patchPath
	g.Cfg.First = g.Opera.firstBlock
	g.Cfg.Last = g.Opera.lastBlock

	patchDb, err := rawdb.NewLevelDBDatabase(g.Cfg.TargetDb, 1024, 100, "profiling", false)
	if err != nil {
		return "", fmt.Errorf("cannot open patch db; %v", err)
	}

	err = CreatePatchClone(g.Cfg, g.AidaDb, patchDb, g.Opera.FirstEpoch, g.Opera.lastEpoch, g.Opera.isNew)
	if err != nil {
		return "", fmt.Errorf("cannot create patch clone; %v", err)
	}

	// metadata
	err = utils.ProcessPatchLikeMetadata(patchDb, g.Cfg.LogLevel, g.Cfg.First, g.Cfg.Last, g.Opera.FirstEpoch,
		g.Opera.lastEpoch, g.Cfg.ChainID, g.Opera.isNew, g.dbHash)
	if err != nil {
		return "", err
	}

	MustCloseDB(patchDb)

	g.Log.Noticef("Printing newly generated patch METADATA:")
	err = PrintMetadata(patchPath)
	if err != nil {
		return "", err
	}

	patchTarName := fmt.Sprintf("%v.tar.gz", patchName)
	patchTarPath := filepath.Join(g.Cfg.Output, patchTarName)

	err = g.createPatchTarGz(patchPath, patchTarName)
	if err != nil {
		return "", fmt.Errorf("unable to create patch tar.gz of %s; %v", patchPath, err)
	}

	g.Log.Noticef("Patch %s generated successfully: %d(%d) - %d(%d) ", patchTarName, g.Cfg.First,
		g.Opera.FirstEpoch, g.Cfg.Last, g.Opera.lastEpoch)

	g.patchTarHash, err = calculateMD5Sum(patchTarPath)
	if err != nil {
		return "", fmt.Errorf("unable to calculate md5sum of %s; %v", patchTarPath, err)
	}

	err = g.updatePatchesJson(patchTarName)
	if err != nil {
		return "", err
	}

	// remove patchFiles
	err = os.RemoveAll(patchPath)
	if err != nil {
		return "", err
	}

	g.Log.Noticef("Patch created successfully. It took: %v", time.Since(start).Round(1*time.Second))
	g.Log.Noticef("Total elapsed time: %v", time.Since(g.start).Round(1*time.Second))

	return patchTarPath, nil
}

// updatePatchesJson with newly acquired patch
func (g *Generator) updatePatchesJson(fileName string) error {
	jsonFilePath := filepath.Join(g.Cfg.Output, patchesJsonName)

	// Load previous JSON
	var patchesJson []utils.PatchJson

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
		patchesJson = make([]utils.PatchJson, 0)
	}

	// Create a new patch object
	newPatch := utils.PatchJson{
		FileName:  fileName,
		FromBlock: g.Cfg.First,
		ToBlock:   g.Cfg.Last,
		FromEpoch: g.Opera.FirstEpoch,
		ToEpoch:   g.Opera.lastEpoch,
		DbHash:    hex.EncodeToString(g.dbHash),
		TarHash:   g.patchTarHash,
		Nightly:   true,
	}

	// Append the new patch to the array
	patchesJson = append(patchesJson, newPatch)

	if err = g.doUpdatePatchesJson(patchesJson, file); err != nil {
		return err
	}

	g.Log.Noticef("Updated %s in %s with new patch: %v", patchesJsonName, jsonFilePath, newPatch)
	return nil
}

// doUpdatePatchesJson with newly acquired patch
func (g *Generator) doUpdatePatchesJson(patchesJson []utils.PatchJson, file *os.File) error {
	// Convert the array to JSON bytes
	jsonBytes, err := json.MarshalIndent(patchesJson, "", "  ")
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
func (g *Generator) createPatchTarGz(filePath string, fileName string) error {
	g.Log.Noticef("Generating compressed %v", fileName)
	err := g.createTarGz(filePath, fileName)
	if err != nil {
		return fmt.Errorf("unable to compress %v; %v", fileName, err)
	}
	return nil
}

// storeMd5sum of patch.tar.gz file
func (g *Generator) storeMd5sum(filePath string) error {
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
func (g *Generator) createTarGz(filePath string, fileName string) interface{} {
	// create a parents of temporary directory
	err := os.MkdirAll(g.Cfg.Output, 0755)
	if err != nil {
		return fmt.Errorf("failed to create %s directory; %s", g.Cfg.Output, err)
	}

	// Create the output file
	file, err := os.Create(filepath.Join(g.Cfg.Output, fileName))
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
func (g *Generator) calculatePatchEnd() error {
	// load last finished epoch to calculate next target
	g.TargetEpoch = g.Opera.FirstEpoch

	// next patch will be at least X epochs large
	if g.Cfg.ChainID == utils.MainnetChainID {
		// mainnet currently takes about 250 epochs per day
		g.TargetEpoch += 250
	} else {
		// generic value - about 3 days on testnet 4002
		g.TargetEpoch += 50
	}

	headEpochNumber, err := utils.FindHeadEpochNumber(g.Cfg.ChainID)
	if err != nil {
		return err
	}

	// if current generator is too far in history, start generation to the current head
	if headEpochNumber > g.TargetEpoch {
		g.TargetEpoch = headEpochNumber
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

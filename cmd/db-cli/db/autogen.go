package db

import (
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// AutoGenCommand generates aida-db patches and handles second opera for event generation
var AutoGenCommand = cli.Command{
	Action: autogen,
	Name:   "autogen",
	Usage:  "autogen generates aida-db periodically",
	Flags: []cli.Flag{
		// TODO minimal epoch length for patch generation
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&utils.DbFlag,
		&utils.GenesisFlag,
		&utils.DbTmpFlag,
		&utils.UpdateBufferSizeFlag,
		&utils.OutputFlag,
		&utils.WorldStateFlag,
		&substate.WorkersFlag,
		&logger.LogLevelFlag,
	},
	Description: `
AutoGen generates aida-db patches and handles second opera for event generation. Generates event file, which is supplied into doGenerations to create aida-db patch.
`,
}

// autogen command is used to record/update aida-db periodically
func autogen(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	g, err := newGenerator(ctx, cfg)
	if err != nil {
		return err
	}

	err = g.opera.init()
	if err != nil {
		return err
	}

	// remove worldstate directory if it was created
	defer func(log *logging.Logger) {
		if cfg.WorldStateDb != "" {
			err = os.RemoveAll(cfg.WorldStateDb)
			if err != nil {
				log.Criticalf("can't remove temporary folder: %v; %v", cfg.WorldStateDb, err)
			}
		}
	}(g.log)

	err = g.calculatePatchEnd()
	if err != nil {
		return err
	}

	g.log.Noticef("Starting substate generation %d - %d", g.opera.lastEpoch+1, g.stopAtEpoch)

	MustCloseDB(g.aidaDb)

	// stop opera to be able to export events
	errCh := startOperaRecording(g.cfg, g.stopAtEpoch)

	// wait for opera recording response
	err, ok := <-errCh
	if ok && err != nil {
		return err
	}
	g.log.Noticef("Opera %v - successfully substates for epoch range %d - %d", g.cfg.Db, g.opera.lastEpoch+1, g.stopAtEpoch)
	a.log.Noticef("Successfully pruned opera: %v", a.cfg.Db)

	// start opera to load new blocks in parallel
	err = startDaemonOpera(a.log)
	if err != nil {
		return err
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
	var (
		err error
	)

	_, err = os.Stat(a.cfg.Db)
	if !os.IsNotExist(err) {
		// opera was already used for generation starting from the next epoch
		// !!! returning number one block greater than actual block
		err = a.opera.getOperaBlockAndEpoch(true)
		if err != nil {
			return false, fmt.Errorf("unable to retrieve epoch of generation opera in path %v; %v", a.cfg.Db, err)
		}
		a.opera.firstEpoch += 1
		a.log.Debugf("Generation will start from: %v", a.opera.firstEpoch)
	}

	a.opera.lastEpoch, err = a.getLastEpochFromRunningOpera()
	if err != nil {
		return false, fmt.Errorf("unable to retrieve epoch of running opera in path %v; %v", a.cfg.OperaDatadir, err)
	}

	// ending generation one epoch sooner to make sure epoch is sealed
	a.opera.lastEpoch -= 1

	if a.opera.firstEpoch > a.opera.lastEpoch {
		return false, nil
	}

	a.log.Debugf("Last available sealed epoch is %v", a.opera.lastEpoch)

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
	patchName := fmt.Sprintf("%v-%v", a.opera.firstEpoch, a.opera.lastEpoch)
	patchPath := filepath.Join(a.cfg.Output, patchName)

	// cfg.AidaDb is now pointing to patch this is needed for Merge function
	a.cfg.AidaDb = patchPath

	// open targetDb
	targetDb, err := rawdb.NewLevelDBDatabase(a.cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return "", fmt.Errorf("cannot open targetDb; %v", err)
	}

	// merge UpdateDb into AidaDb
	err = a.mergePatch(targetDb)
	if err != nil {
		return "", fmt.Errorf("unable to merge into patch; %v", err)
	}

	a.log.Notice("Patch metadata")

	// metadata
	err = utils.ProcessPatchLikeMetadata(targetDb, a.cfg.LogLevel, a.cfg.First, a.cfg.Last, a.opera.firstEpoch,
		a.opera.lastEpoch, a.cfg.ChainID, a.opera.isNew)
	if err != nil {
		return "", err
	}

	MustCloseDB(targetDb)

	patchTarName := fmt.Sprintf("%v.tar.gz", patchName)
	patchTarPath := filepath.Join(a.cfg.Output, patchTarName)

	err = a.createPatchTarGz(patchPath, patchTarName)
	if err != nil {
		return "", fmt.Errorf("unable to create patch tar.gz of %s; %v", patchPath, err)
	}

	a.log.Noticef("Patch %s generated successfully: %d(%d) - %d(%d) ", patchTarName, a.cfg.First,
		a.opera.firstEpoch, a.cfg.Last, a.opera.lastEpoch)

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
func (a *automator) mergePatch(targetDb ethdb.Database) error {

	sourceDbPaths := []string{a.cfg.SubstateDb, a.cfg.UpdateDb, a.cfg.DeletionDb}

	dbs, err := openSourceDatabases(sourceDbPaths)
	if err != nil {
		return err
	}

	m := newMerger(a.cfg, targetDb, dbs, sourceDbPaths, nil)

	return m.merge()
}

// updatePatchesJson with newly acquired patch
func (a *automator) updatePatchesJson(fileName string) error {
	jsonFilePath := filepath.Join(a.cfg.Output, patchesJsonName)
	var patchesJson []map[string]string

	// reopen aida-db
	g.aidaDb, err = rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		log.Fatalf("cannot create new db; %v", err)
		return err
	}
	substate.SetSubstateDbBackend(g.aidaDb)

	err = g.opera.getOperaBlockAndEpoch(false)
	if err != nil {
		return err
	}

	return g.Generate()
}

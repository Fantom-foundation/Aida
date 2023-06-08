package db

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"

	"github.com/Fantom-foundation/Aida/cmd/db-cli/flags"
	"github.com/Fantom-foundation/Aida/cmd/substate-cli/replay"
	"github.com/Fantom-foundation/Aida/cmd/updateset-cli/updateset"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// GenerateCommand data structure for the replay app
var GenerateCommand = cli.Command{
	Action: gen,
	Name:   "gen",
	Usage:  "generates aida-db from given events",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.DbFlag,
		&utils.GenesisFlag,
		&utils.KeepDbFlag,
		&utils.CompactDbFlag,
		&utils.DbTmpFlag,
		&utils.UpdateBufferSizeFlag,
		&utils.ChannelBufferSizeFlag,
		&utils.ChainIDFlag,
		&utils.CacheFlag,
		&logger.LogLevelFlag,
		&flags.SkipMetadata,
	},
	Description: `
The db generate command requires events as an argument:
<events>

<events> are fed into the opera database (either existing or genesis needs to be specified), processing them generates updated aida-db.`,
}

const (
	// default updateSet interval
	updateSetInterval = 1_000_000
	// number of lines which are kept in memory in case command fails
	commandOutputLimit = 50
)

type generator struct {
	cfg       *utils.Config
	log       *logging.Logger
	aidaDb    ethdb.Database
	aidaDbTmp string
	opera     *aidaOpera
}

func gen(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.EventArg)
	if err != nil {
		return fmt.Errorf("cannot create config %v", err)
	}

	aidaDbTmp, err := prepareDbDirs(cfg)
	if err != nil {
		return fmt.Errorf("cannot create config %v", err)
	}

	cfg.Workers = substate.WorkersFlag.Value

	g := newGenerator(ctx, cfg, aidaDbTmp)

	defer MustCloseDB(g.aidaDb)

	if g.cfg.AidaDb == "" {
		return fmt.Errorf("you need to specify where you want aida-db to save (--aida-db)")
	}

	return g.Generate()
}

func newGenerator(ctx *cli.Context, cfg *utils.Config, aidaDbTmp string) *generator {
	db, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		log.Fatalf("cannot create new db; %v", err)
	}

	log := logger.NewLogger("AidaDb-Generator", cfg.LogLevel)

	return &generator{
		cfg:       cfg,
		log:       log,
		aidaDbTmp: aidaDbTmp,
		opera:     newAidaOpera(ctx, cfg, log),
		aidaDb:    db,
	}
}

// Generate is used to record/update aida-db
func (g *generator) Generate() error {
	var err error

	if err = g.opera.init(); err != nil {
		return err
	}

	if err = g.processSubstate(); err != nil {
		return err
	}

	if err = g.processDeletedAccounts(); err != nil {
		return err
	}

	if err = g.processUpdateSet(); err != nil {
		return err
	}

	g.openAidaDb()

	processGenLikeMetadata(g.aidaDb, g.cfg.LogLevel, g.opera.firstBlock, g.opera.lastBlock, g.opera.firstEpoch, g.opera.lastEpoch, g.cfg.ChainID)

	if !g.cfg.KeepDb {
		err = os.RemoveAll(g.aidaDbTmp)
		if err != nil {
			return err
		}
	}

	g.log.Noticef("AidaDb %v generation done", g.cfg.AidaDb)

	return nil
}

// processSubstate loads events into the opera, whilst recording substates and then merges it into AidaDb
func (g *generator) processSubstate() error {
	var (
		err error
		cmd *exec.Cmd
	)

	_, err = os.Stat(g.cfg.Events)
	if os.IsNotExist(err) {
		return fmt.Errorf("supplied events file %s doesn't exist", g.cfg.Events)
	}

	g.log.Noticef("Starting Substate recording from %v", g.cfg.Events)

	cmd = exec.Command("opera", "--datadir", g.cfg.Db, "--cache", strconv.Itoa(g.cfg.Cache),
		"import", "events", "--recording", "--substate-db", g.cfg.SubstateDb, g.cfg.Events)

	err = runCommand(cmd, nil, g.log)
	if err != nil {
		// remove empty substateDb
		return fmt.Errorf("cannot import events; %v", err)
	}

	// retrieve block the opera was iterated onto
	g.opera.lastBlock, g.opera.lastEpoch, err = GetOperaBlockAndEpoch(g.cfg)
	if err != nil {
		return fmt.Errorf("cannot get last opera block and epoch; %v", err)
	}

	if g.opera.firstBlock >= g.opera.lastBlock {
		return fmt.Errorf("supplied events didn't produce any new blocks")
	}

	g.log.Infof("Substates generated for %v - %v", g.opera.firstBlock, g.opera.lastBlock)

	g.log.Notice("Merging SubstateDb into AidaDb...")

	if err = g.merge(g.cfg.SubstateDb); err != nil {
		return err
	}

	// merge was successful - set new path to substateDb
	g.log.Notice("SubstateDb merged successfully")
	g.cfg.SubstateDb = g.cfg.AidaDb

	return nil
}

// processDeletedAccounts invokes DeletedAccounts generation and then merges it into AidaDb
func (g *generator) processDeletedAccounts() error {
	var err error

	g.log.Noticef("Generating DeletionDb...")

	err = replay.GenDeletedAccountsAction(g.cfg)
	if err != nil {
		return fmt.Errorf("cannot generate deleted accounts; %v", err)
	}

	g.log.Noticef("Deleted accounts generated successfully")

	g.log.Notice("Merging DeletionDb into AidaDb...")

	if err = g.merge(g.cfg.DeletionDb); err != nil {
		return err
	}

	// merge was successful - set new path to deletionDb
	g.log.Notice("DeletionDb merged successfully")
	g.cfg.DeletionDb = g.cfg.AidaDb

	return nil
}

// processUpdateSet invokes UpdateSet generation and then merges it into AidaDb
func (g *generator) processUpdateSet() error {
	var (
		updateDb           *substate.UpdateDB
		err                error
		nextUpdateSetStart uint64
	)

	updateDb, err = substate.OpenUpdateDB(g.cfg.AidaDb)
	if err != nil {
		return err
	}

	// set first block
	nextUpdateSetStart = updateDb.GetLastKey() + 1
	err = updateDb.Close()
	if err != nil {
		return errors.New("cannot close updateDb")
	}

	if nextUpdateSetStart > 1 {
		g.log.Infof("Previous UpdateSet found - generating from %v", nextUpdateSetStart)
	}

	g.log.Notice("Generating UpdateDb...")

	err = updateset.GenUpdateSet(g.cfg, nextUpdateSetStart, updateSetInterval)
	if err != nil {
		return fmt.Errorf("cannot generate update-db")
	}

	g.log.Notice("UpdateDb generated successfully")
	g.log.Notice("Merging UpdateDb into AidaDb...")

	if err = g.merge(g.cfg.UpdateDb); err != nil {
		return err
	}

	g.log.Notice("UpdateDB merged successfully")

	// merge was successful - set new path to updateDb
	g.cfg.UpdateDb = g.cfg.AidaDb

	return nil

}

// merge sole dbs created in generation into AidaDb
func (g *generator) merge(pathToDb string) error {
	// open targetDb
	targetDb, err := rawdb.NewLevelDBDatabase(g.cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open targetDb; %v", err)
	}

	// open sourceDb
	sourceDb, err := rawdb.NewLevelDBDatabase(g.cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return err
	}

	m := newMerger(g.cfg, targetDb, []ethdb.Database{sourceDb}, []string{pathToDb})

	return m.merge()
}

func (g *generator) openAidaDb() {
	g.aidaDb, _ = rawdb.NewLevelDBDatabase(g.cfg.AidaDb, 1024, 100, "profiling", false)
	return
}

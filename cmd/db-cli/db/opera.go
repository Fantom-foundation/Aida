package db

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/Fantom-foundation/Aida/cmd/worldstate-cli/state"
	"github.com/Fantom-foundation/Aida/utils"
	wsOpera "github.com/Fantom-foundation/Aida/world-state/db/opera"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// aidaOpera represents running opera as a subprocess
type aidaOpera struct {
	firstBlock, lastBlock uint64
	firstEpoch, lastEpoch uint64
	ctx                   *cli.Context
	cfg                   *utils.Config
	log                   *logging.Logger
	isNew                 bool
}

// newAidaOpera returns new instance of Opera
func newAidaOpera(ctx *cli.Context, cfg *utils.Config, log *logging.Logger) *aidaOpera {
	return &aidaOpera{
		ctx: ctx,
		cfg: cfg,
		log: log,
	}
}

// init aidaOpera by executing command to start (and stop) opera and preparing dump context
func (opera *aidaOpera) init() error {
	var err error

	_, err = os.Stat(opera.cfg.Db)
	if os.IsNotExist(err) {
		opera.isNew = true

		opera.log.Noticef("Initialising opera from genesis")

		// previous opera database isn't used - generate new one from genesis
		err = opera.initFromGenesis()
		if err != nil {
			return fmt.Errorf("cannot init opera from gensis; %v", err)
		}

		// create tmpDir for worldstate
		var tmpDir string
		tmpDir, err = createTmpDir(opera.cfg)
		if err != nil {
			return fmt.Errorf("cannot create tmp dir; %v", err)
		}
		opera.cfg.WorldStateDb = filepath.Join(tmpDir, "worldstate")

		// dumping the MPT into world state
		if err = opera.prepareDumpCliContext(); err != nil {
			return fmt.Errorf("cannot prepare dump; %v", err)
		}
	}

	// get first block and epoch
	// running this command before starting opera results in getting first block and epoch on which opera starts
	err = opera.getOperaBlockAndEpoch(true)
	if err != nil {
		return fmt.Errorf("cannot retrieve block from existing opera database %v; %v", opera.cfg.Db, err)
	}

	// when initializing fresh opera, the block returned by it is -1 because it has not generated any blocks yet
	// for this to work correctly, we need to up operas first block by one
	if opera.isNew {
		opera.firstBlock++
	}

	opera.log.Noticef("Opera is starting at block: %v", opera.firstBlock)

	// starting generation one block later
	opera.cfg.First = opera.firstBlock + 1
	opera.firstEpoch += 1
	return nil
}

func createTmpDir(cfg *utils.Config) (string, error) {
	if cfg.DbTmp != "" {
		// create a parents of temporary directory
		err := os.MkdirAll(cfg.DbTmp, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create %s directory; %s", cfg.DbTmp, err)
		}
	}

	//fName := fmt.Sprintf("%v/%v-%v", cfg.DbTmp, "tmp_aida_db_*", rand.Int())
	// create a temporary working directory
	fName, err := os.MkdirTemp(cfg.DbTmp, "aida_db_tmp_*")
	if err != nil {
		return "", fmt.Errorf("failed to create a temporary directory. %v", err)
	}

	return fName, nil
}

// initFromGenesis file
func (opera *aidaOpera) initFromGenesis() error {
	cmd := exec.Command("opera", "--datadir", opera.cfg.Db, "--genesis", opera.cfg.Genesis,
		"--exitwhensynced.epoch=0", "--cache", strconv.Itoa(opera.cfg.Cache), "--db.preset=legacy-ldb", "--maxpeers=0")

	err := runCommand(cmd, nil, opera.log)
	if err != nil {
		return fmt.Errorf("load opera genesis; %v", err.Error())
	}

	return nil
}

// rollbackToEpoch file TODO should be part of future autogen recovery
func (opera *aidaOpera) rollbackToEpoch() error {
	//cmd := exec.Command("opera", "--datadir", opera.cfg.Db, "--genesis", opera.cfg.Genesis,
	//	"--exitwhensynced.epoch=0", "--cache", strconv.Itoa(opera.cfg.Cache), "--db.preset=legacy-ldb", "--maxpeers=0", "db", "heal", "--experimental")
	//
	//err := runCommand(cmd, nil, opera.log)
	//if err != nil {
	//	return fmt.Errorf("load opera genesis; %v", err.Error())
	//}
	//
	return nil
}

// getOperaBlockAndEpoch retrieves current block of opera head
func (opera *aidaOpera) getOperaBlockAndEpoch(isFirst bool) error {
	operaPath := filepath.Join(opera.cfg.Db, "/chaindata/leveldb-fsh/")
	store, err := wsOpera.Connect("ldb", operaPath, "main")
	if err != nil {
		return err
	}
	defer wsOpera.MustCloseStore(store)

	_, blockNumber, epochNumber, err := wsOpera.LatestStateRoot(store)
	if err != nil {
		return fmt.Errorf("state root not found; %v", err)
	}

	if blockNumber < 1 {
		return fmt.Errorf("opera; block number not found; %v", err)
	}

	// we are assuming that we are at brink of epochs
	// in this special case epochNumber is already one number higher
	epochNumber -= 1

	// todo check ifNew then fb + 1
	if isFirst {
		// opera returns block off by one
		opera.firstBlock = blockNumber
		opera.firstEpoch = epochNumber
	} else {
		opera.lastBlock = blockNumber
		opera.lastEpoch = epochNumber
	}

	return nil
}

// prepareDumpCliContext
func (opera *aidaOpera) prepareDumpCliContext() error {
	// TODO rewrite
	tmpSaveDbPath := opera.cfg.Db
	opera.cfg.Db = filepath.Join(opera.cfg.Db, "chaindata/leveldb-fsh/")
	opera.cfg.DbVariant = "ldb"
	err := state.DumpState(opera.ctx, opera.cfg)
	if err != nil {
		return err
	}
	opera.cfg.Db = tmpSaveDbPath
	return nil
	//flagSet := flag.NewFlagSet("", 0)
	//flagSet.String(utils.WorldStateFlag.Name, opera.cfg.WorldStateDb, "")
	//flagSet.String(utils.DbFlag.Name, opera.cfg.Db+"/chaindata/leveldb-fsh/", "")
	//flagSet.String(utils.StateDbVariantFlag.Name, "ldb", "")
	//flagSet.String(utils.SourceTableNameFlag.Name, utils.SourceTableNameFlag.Value, "")
	//flagSet.String(utils.TrieRootHashFlag.Name, utils.TrieRootHashFlag.Value, "")
	//flagSet.Int(substate.WorkersFlag.Name, substate.WorkersFlag.Value, "")
	//flagSet.Uint64(utils.TargetBlockFlag.Name, utils.TargetBlockFlag.Value, "")
	//flagSet.Int(utils.ChainIDFlag.Name, opera.cfg.ChainID, "")
	//flagSet.String(logger.LogLevelFlag.Name, opera.cfg.LogLevel, "")
	//
	//ctx := cli.NewContext(cli.NewApp(), flagSet, nil)
	//
	//err := ctx.Set(utils.DbFlag.Name, opera.cfg.Db+"/chaindata/leveldb-fsh/")
	//if err != nil {
	//	return err
	//}
	//command := &cli.Command{Name: state.CmdDumpState.Name}
	//ctx.Command = command
	//ctx.Command.Flags = state.CmdDumpState.Flags
	//
	//return state.DumpState(ctx)
}

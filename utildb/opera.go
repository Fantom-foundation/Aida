package utildb

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// aidaOpera represents running opera as a subprocess
type aidaOpera struct {
	firstBlock, lastBlock uint64
	FirstEpoch, lastEpoch uint64
	ctx                   *cli.Context
	cfg                   *utils.Config
	log                   logger.Logger
	isNew                 bool
}

// newAidaOpera returns new instance of Opera
func newAidaOpera(ctx *cli.Context, cfg *utils.Config, log logger.Logger) *aidaOpera {
	return &aidaOpera{
		ctx: ctx,
		cfg: cfg,
		log: log,
	}
}

// init aidaOpera by executing command to start (and stop) opera and preparing dump context
func (opera *aidaOpera) init() error {
	// TODO resolve dependencies
	panic("feature not supported")
	/*
		var err error

		_, err = os.Stat(opera.cfg.OperaDb)
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
			return fmt.Errorf("cannot retrieve block from existing opera database %v; %v", opera.cfg.OperaDb, err)
		}

		opera.log.Noticef("Opera block from last run is: %v", opera.firstBlock)

		// starting generation one block later
		opera.firstBlock += 1
		opera.FirstEpoch += 1
	*/
	return nil
}

func createTmpDir(cfg *utils.Config) (string, error) {
	// create a temporary working directory
	fName, err := os.MkdirTemp(cfg.DbTmp, "aida_db_tmp_*")
	if err != nil {
		return "", fmt.Errorf("failed to create a temporary directory. %v", err)
	}

	return fName, nil
}

// initFromGenesis file
func (opera *aidaOpera) initFromGenesis() error {
	cmd := exec.Command(getOperaBinary(opera.cfg), "--datadir", opera.cfg.OperaDb, "--genesis", opera.cfg.Genesis,
		"--exitwhensynced.epoch=0", "--cache", strconv.Itoa(opera.cfg.Cache), "--db.preset=legacy-ldb", "--maxpeers=0")

	err := runCommand(cmd, nil, nil, opera.log)
	if err != nil {
		return fmt.Errorf("load opera genesis; %v", err.Error())
	}

	return nil
}

// getOperaBlockAndEpoch retrieves current block of opera head
func (opera *aidaOpera) getOperaBlockAndEpoch(isFirst bool) error {
	// TODO resolve dependencies
	panic("feature not supported")
	/*
		operaPath := filepath.Join(opera.cfg.OperaDb, "/chaindata/leveldb-fsh/")
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

		if isFirst {
			opera.firstBlock = blockNumber
			opera.FirstEpoch = epochNumber
		} else {
			opera.lastBlock = blockNumber
			opera.lastEpoch = epochNumber
		}
	*/
	return nil
}

// prepareDumpCliContext prepares cli context for dumping the MPT into world state
func (opera *aidaOpera) prepareDumpCliContext() error {
	// TODO: resolve dependencies
	/*
		// Dump uses cfg.OperaDb and overwrites it therefore the original value needs to be saved and retrieved after DumpState
		tmpSaveDbPath := opera.cfg.OperaDb
		defer func() {
			opera.cfg.OperaDb = tmpSaveDbPath
		}()
		opera.cfg.OperaDb = filepath.Join(opera.cfg.OperaDb, "chaindata/leveldb-fsh/")
		opera.cfg.DbVariant = "ldb"
		err := state.DumpState(opera.ctx, opera.cfg)
		if err != nil {
			return err
		}
	*/
	return nil
}

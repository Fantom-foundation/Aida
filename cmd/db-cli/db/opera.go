package db

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/Fantom-foundation/Aida/cmd/worldstate-cli/state"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
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
func (o *aidaOpera) init() error {
	var err error

	_, err = os.Stat(o.cfg.Db)
	if os.IsNotExist(err) {
		o.log.Noticef("Initialising opera from genesis")

		// previous opera database isn't used - generate new one from genesis
		err = o.initFromGenesis()
		if err != nil {
			return fmt.Errorf("cannot init opera from gensis; %v", err)
		}
	}

	// dumping the MPT into world state
	if err = o.prepareDumpCliContext(); err != nil {
		return fmt.Errorf("cannot prepare dump; %v", err)
	}

	// get first block and epoch
	// running this command before starting opera results in getting first block and epoch on which opera starts
	o.firstBlock, o.firstEpoch, err = GetOperaBlockAndEpoch(o.cfg)
	if err != nil {
		return fmt.Errorf("cannot retrieve block from existing opera database %v; %v", o.cfg.Db, err)
	}

	o.log.Noticef("Opera is starting at block: %v", o.firstBlock)

	// starting generation one block later
	o.cfg.First = o.firstBlock + 1
	return nil
}

// initFromGenesis file
func (o *aidaOpera) initFromGenesis() error {
	cmd := exec.Command("opera", "--datadir", o.cfg.Db, "--genesis", o.cfg.Genesis,
		"--exitwhensynced.epoch=0", "--cache", strconv.Itoa(o.cfg.Cache), "--db.preset=legacy-ldb", "--maxpeers=0")

	err := runCommand(cmd, nil, o.log)
	if err != nil {
		return fmt.Errorf("load opera genesis; %v", err.Error())
	}

	return nil
}

// prepareDumpCliContext
func (o *aidaOpera) prepareDumpCliContext() error {
	flagSet := flag.NewFlagSet("", 0)
	flagSet.String(utils.WorldStateFlag.Name, o.cfg.WorldStateDb, "")
	flagSet.String(utils.DbFlag.Name, o.cfg.Db+"/chaindata/leveldb-fsh/", "")
	flagSet.String(utils.StateDbVariantFlag.Name, "ldb", "")
	flagSet.String(utils.SourceTableNameFlag.Name, utils.SourceTableNameFlag.Value, "")
	flagSet.String(utils.TrieRootHashFlag.Name, utils.TrieRootHashFlag.Value, "")
	flagSet.Int(substate.WorkersFlag.Name, substate.WorkersFlag.Value, "")
	flagSet.Uint64(utils.TargetBlockFlag.Name, utils.TargetBlockFlag.Value, "")
	flagSet.Int(utils.ChainIDFlag.Name, o.cfg.ChainID, "")
	flagSet.String(logger.LogLevelFlag.Name, o.cfg.LogLevel, "")

	ctx := cli.NewContext(cli.NewApp(), flagSet, nil)

	err := ctx.Set(utils.DbFlag.Name, o.cfg.Db+"/chaindata/leveldb-fsh/")
	if err != nil {
		return err
	}
	command := &cli.Command{Name: state.CmdDumpState.Name}
	ctx.Command = command

	return state.DumpState(ctx)
}

// generateEvents from given event argument
func (o *aidaOpera) generateEvents(firstEpoch, lastEpoch uint64, aidaDbTmp string) error {
	eventsFile := fmt.Sprintf("events-%v-%v", firstEpoch, lastEpoch)
	o.cfg.Events = filepath.Join(aidaDbTmp, eventsFile)

	o.log.Debugf("Generating events from %v to %v into %v", firstEpoch, lastEpoch, o.cfg.Events)

	cmd := exec.Command(fmt.Sprintf("opera --datadir %v export events %v %v %v", o.cfg.OperaDatadir, o.cfg.Events, firstEpoch, lastEpoch))
	err := runCommand(cmd, nil, o.log)
	if err != nil {
		return fmt.Errorf("retrieve last opera epoch trough ipc; %v", err.Error())
	}

	return nil
}

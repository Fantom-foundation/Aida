package db

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	substate "github.com/Fantom-foundation/Substate"

	"github.com/Fantom-foundation/Aida/cmd/substate-cli/replay"
	"github.com/Fantom-foundation/Aida/cmd/updateset-cli/updateset"
	"github.com/Fantom-foundation/Aida/cmd/worldstate-cli/flags"
	"github.com/Fantom-foundation/Aida/cmd/worldstate-cli/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/world-state/db/opera"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

const updateSetInterval = 1000000

// UpdateCommand data structure for the replay app
var UpdateCommand = cli.Command{
	Action: Update,
	Name:   "update",
	Usage:  "generates aida-db from given events",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.DbFlag,
		&utils.GenesisFlag,
		&utils.DeleteSourceDbsFlag,
		&utils.DbTmpFlag,
		&utils.ChainIDFlag,
		&utils.LogLevelFlag,
	},
	Description: `
The db update command requires events as an argument:
<events>

<events> are fed into the opera database (either existing or genesis needs to be specified), processing them generates updated aida-db.`,
}

// Update command is used to record/update aida-db
func Update(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.EventArg)
	if argErr != nil {
		return argErr
	}

	log := utils.NewLogger(cfg.LogLevel, "Update")

	updateConfigFlags(cfg, log)

	usedGenesis, err := prepareOpera(cfg, log)
	if err != nil {
		return err
	}

	err = recordSubstate(cfg, log)
	if err != nil {
		return err
	}

	err = genDeletedAccounts(cfg, log)
	if err != nil {
		return err
	}

	err = genUpdateSet(cfg, usedGenesis, log)
	if err != nil {
		return err
	}

	// merge generated databases into aida-db
	err = Merge(cfg, []string{cfg.SubstateDb, cfg.UpdateDb, cfg.DeletionDb})
	if err != nil {
		return err
	}

	// delete source databases
	if cfg.DeleteSourceDbs {
		err = os.RemoveAll(cfg.WorldStateDb)
		if err != nil {
			return err
		}
		log.Infof("deleted: %s", cfg.WorldStateDb)
	}

	log.Infof("Aida-db updated successfully from block %v to block %v\n", cfg.First, cfg.Last)

	return err
}

// prepareOpera confirms that the opera is initialized
func prepareOpera(cfg *utils.Config, log *logging.Logger) (bool, error) {
	usedGenesis := false
	_, err := os.Stat(cfg.Db)
	if os.IsNotExist(err) {
		log.Noticef("Initialising opera from genesis")
		// previous opera database isn't used - generate new one from genesis
		err := initOperaFromGenesis(cfg)
		if err != nil {
			return false, fmt.Errorf("aida-db; Error: %v", err)
		}
		usedGenesis = true
	}
	cfg.First, err = getOperaBlock(cfg)
	if err != nil {
		return false, fmt.Errorf("couldn't retrieve block from existing opera database %v ; Error: %v", cfg.Db, err)
	}

	log.Noticef("Opera is starting at block: %v", cfg.First)

	return usedGenesis, nil
}

// updateConfigFlags updates config for flags required in invoked generation commands
// these flags are not expected from user, so we need to specify them for the generation process
func updateConfigFlags(cfg *utils.Config, log *logging.Logger) {
	if cfg.DbTmp == "" {
		log.Fatalf("--%v needs to be specified", utils.DbTmpFlag.Name)
	}

	cfg.DeletionDb = cfg.DbTmp + "/deletion"
	cfg.SubstateDb = cfg.DbTmp + "/substate"
	cfg.UpdateDb = cfg.DbTmp + "/update"
	cfg.WorldStateDb = cfg.DbTmp + "/worldstate"
	cfg.Workers = substate.WorkersFlag.Value
}

// getOperaBlock retrieves current block of opera head
func getOperaBlock(cfg *utils.Config) (uint64, error) {
	store, err := opera.Connect("ldb", cfg.Db+"/chaindata/leveldb-fsh/", "main")
	if err != nil {
		return 0, err
	}
	defer opera.MustCloseStore(store)

	_, blockNumber, err := opera.LatestStateRoot(store)
	if err != nil {
		return 0, fmt.Errorf("state root not found; %v", err)
	}

	if blockNumber < 1 {
		return 0, fmt.Errorf("opera; block number not found; %v", err)
	}
	return blockNumber, nil
}

// genUpdateSet invokes UpdateSet generation
func genUpdateSet(cfg *utils.Config, usedGenesis bool, log *logging.Logger) error {
	log.Noticef("UpdateSet generation")
	return updateset.GenUpdateSet(cfg, updateSetInterval)
}

// genDeletedAccounts invokes DeletedAccounts generation
func genDeletedAccounts(cfg *utils.Config, log *logging.Logger) error {
	log.Noticef("Deleted generation")
	err := replay.GenDeletedAccountsAction(cfg)
	if err != nil {
		return fmt.Errorf("DelAccounts; %v", err)
	}
	return nil
}

// recordSubstate loads events into the opera, whilst recording substates
func recordSubstate(cfg *utils.Config, log *logging.Logger) error {
	_, err := os.Stat(cfg.Events)
	if os.IsNotExist(err) {
		return fmt.Errorf("supplied events file %s doesn't exist", cfg.Events)
	}

	cmd := exec.Command("opera", "--datadir", cfg.Db, "--gcmode=light", "--db.preset=legacy-ldb", "import", "events", "--recording", "--substatedir", cfg.SubstateDb, cfg.Events)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return fmt.Errorf("load opera genesis; %v", err.Error())
	}

	// retrieve block the opera was iterated into
	cfg.Last, err = getOperaBlock(cfg)
	if err != nil {
		return fmt.Errorf("getOperaBlock last; %v", err)
	}
	if (cfg.Last - cfg.First) < 1 {
		return fmt.Errorf("supplied events didn't produce any new blocks")
	}

	log.Noticef("Substates generated for %v - %v", cfg.First, cfg.Last)

	return nil
}

// initOperaFromGenesis prepares opera by loading genesis
func initOperaFromGenesis(cfg *utils.Config) error {
	cmd := exec.Command("opera", "--datadir", cfg.Db, "--genesis", cfg.Genesis, "--exitwhensynced.epoch=0", "--db.preset=legacy-ldb", "--maxpeers=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return fmt.Errorf("load opera genesis; %v", err.Error())
	}

	// dumping the MPT into world state
	dumpCli, err := prepareDumpCliContext(cfg)
	if err != nil {
		return err
	}
	err = state.DumpState(dumpCli)
	if err != nil {
		return fmt.Errorf("dumpState; %v", err)
	}
	return nil
}

// TODO rewrite after dump is using the config then pass cfg directly to the dump function
func prepareDumpCliContext(cfg *utils.Config) (*cli.Context, error) {
	flagSet := flag.NewFlagSet("", 0)
	flagSet.String(flags.StateDBPath.Name, cfg.WorldStateDb, "")
	flagSet.String(flags.SourceDBPath.Name, cfg.Db+"/chaindata/leveldb-fsh/", "")
	flagSet.String(flags.SourceDBType.Name, flags.SourceDBType.Value, "")
	flagSet.String(flags.SourceTableName.Name, flags.SourceTableName.Value, "")
	flagSet.String(flags.TrieRootHash.Name, flags.TrieRootHash.Value, "")
	flagSet.Int(flags.Workers.Name, flags.Workers.Value, "")
	flagSet.Uint64(flags.TargetBlock.Name, flags.TargetBlock.Value, "")

	ctx := cli.NewContext(cli.NewApp(), flagSet, nil)

	err := ctx.Set(flags.SourceDBPath.Name, cfg.Db+"/chaindata/leveldb-fsh/")
	if err != nil {
		return nil, err
	}
	command := &cli.Command{Name: state.CmdDumpState.Name}
	ctx.Command = command

	return ctx, nil
}

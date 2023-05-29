package db

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/Fantom-foundation/Aida/cmd/db-cli/flags"
	"github.com/Fantom-foundation/Aida/cmd/substate-cli/replay"
	"github.com/Fantom-foundation/Aida/cmd/updateset-cli/updateset"
	"github.com/Fantom-foundation/Aida/cmd/worldstate-cli/state"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/world-state/db/opera"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// default updateSet interval
const updateSetInterval = 1_000_000

// GenerateCommand data structure for the replay app
var GenerateCommand = cli.Command{
	Action: generate,
	Name:   "generate",
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

// generate prepares config for Generate
func generate(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.EventArg)
	if argErr != nil {
		return argErr
	}

	log := logger.NewLogger(cfg.LogLevel, "Generate")

	aidaDbTmp, err := prepare(cfg)
	if err != nil {
		return err
	}

	if !cfg.KeepDb {
		defer func() {
			err = os.RemoveAll(aidaDbTmp)
			if err != nil {
				panic(err)
			}
		}()
	}

	_, err = Generate(ctx, cfg, log)
	if err != nil {
		return err
	}

	return nil
}

// Generate is used to record/update aida-db
func Generate(ctx *cli.Context, cfg *utils.Config, log *logging.Logger) (*MetadataInfo, error) {
	mdi := new(MetadataInfo)
	mdi.dbType = genType
	// todo extract chainid from opera
	err := prepareOpera(ctx, cfg, log, mdi)
	if err != nil {
		return nil, err
	}

	mdi.chainId = cfg.ChainID

	err = recordSubstate(cfg, log, mdi)
	if err != nil {
		return nil, err
	}

	err = genDeletedAccounts(cfg, log, mdi)
	if err != nil {
		return nil, err
	}

	err = genUpdateSet(cfg, log, mdi)
	if err != nil {
		return nil, err
	}

	log.Noticef("Aida-db updated from block %v to %v", cfg.First-1, cfg.Last)

	return nil, nil
}

// prepareOpera confirms that the opera is initialized
func prepareOpera(ctx *cli.Context, cfg *utils.Config, log *logging.Logger, mdi *MetadataInfo) error {
	_, err := os.Stat(cfg.Db)
	if os.IsNotExist(err) {
		if ctx.String(utils.ChainIDFlag.Name) == "" {
			return fmt.Errorf("you need to specify chain-id when creating new aida-db (--%v)", utils.ChainIDFlag.Name)
		}

		log.Noticef("Initialising opera from genesis")
		// previous opera database isn't used - generate new one from genesis
		err = initOperaFromGenesis(cfg, log)
		if err != nil {
			return fmt.Errorf("aida-db; Error: %v", err)
		}
	}
	lastOperaBlock, firstEpoch, err := GetOperaBlock(cfg)
	if err != nil {
		return fmt.Errorf("couldn't retrieve block from existing opera database %v ; Error: %v", cfg.Db, err)
	}

	mdi.firstEpoch = firstEpoch

	log.Noticef("Opera is starting at block: %v", lastOperaBlock)

	//starting generation one block later
	cfg.First = lastOperaBlock + 1
	return nil
}

// prepare updates config for flags required in invoked generation commands
// these flags are not expected from user, so we need to specify them for the generation process
func prepare(cfg *utils.Config) (string, error) {
	if cfg.DbTmp != "" {
		// create a parents of temporary directory
		err := os.MkdirAll(cfg.DbTmp, 0700)
		if err != nil {
			return "", fmt.Errorf("failed to create %s directory; %s", cfg.DbTmp, err)
		}
	}
	//create a temporary working directory
	aidaDbTmp, err := ioutil.TempDir(cfg.DbTmp, "aida_db_tmp_*")
	if err != nil {
		return "", fmt.Errorf("failed to create a temporary directory. %v", err)
	}

	loadSourceDBPaths(cfg, aidaDbTmp)

	cfg.Workers = substate.WorkersFlag.Value

	return aidaDbTmp, nil
}

// loadSourceDBPaths initializes paths to source databases
func loadSourceDBPaths(cfg *utils.Config, aidaDbTmp string) {
	cfg.DeletionDb = filepath.Join(aidaDbTmp, "deletion")
	cfg.SubstateDb = filepath.Join(aidaDbTmp, "substate")
	cfg.UpdateDb = filepath.Join(aidaDbTmp, "update")
	cfg.WorldStateDb = filepath.Join(aidaDbTmp, "worldstate")
}

// GetOperaBlock retrieves current block of opera head
func GetOperaBlock(cfg *utils.Config) (uint64, uint64, error) {
	operaPath := filepath.Join(cfg.Db, "/chaindata/leveldb-fsh/")
	store, err := opera.Connect("ldb", operaPath, "main")
	if err != nil {
		return 0, 0, err
	}
	defer opera.MustCloseStore(store)

	_, blockNumber, epochNumber, err := opera.LatestStateRoot(store)
	if err != nil {
		return 0, 0, fmt.Errorf("state root not found; %v", err)
	}

	if blockNumber < 1 {
		return 0, 0, fmt.Errorf("opera; block number not found; %v", err)
	}
	return blockNumber, epochNumber, nil
}

// genUpdateSet invokes UpdateSet generation
func genUpdateSet(cfg *utils.Config, log *logging.Logger, mdi *MetadataInfo) error {
	db, err := substate.OpenUpdateDB(cfg.AidaDb)
	if err != nil {
		return err
	}
	// set first block
	nextUpdateSetStart := db.GetLastKey() + 1
	err = db.Close()
	if err != nil {
		return err
	}

	if nextUpdateSetStart > 1 {
		log.Infof("Previous UpdateSet found generating from %v", nextUpdateSetStart)
	}

	log.Noticef("UpdateSet generation")
	err = updateset.GenUpdateSet(cfg, nextUpdateSetStart, updateSetInterval)
	if err != nil {
		return err
	}

	// merge UpdateDb into AidaDb
	err = Merge(cfg, []string{cfg.UpdateDb}, mdi)
	if err != nil {
		return err
	}
	cfg.UpdateDb = cfg.AidaDb

	return nil
}

// genDeletedAccounts invokes DeletedAccounts generation
func genDeletedAccounts(cfg *utils.Config, log *logging.Logger, mdi *MetadataInfo) error {
	log.Noticef("Deleted generation")
	err := replay.GenDeletedAccountsAction(cfg)
	if err != nil {
		return fmt.Errorf("DelAccounts; %v", err)
	}

	// merge DeletionDb into AidaDb
	err = Merge(cfg, []string{cfg.DeletionDb}, mdi)
	if err != nil {
		return err
	}
	cfg.DeletionDb = cfg.AidaDb

	return nil
}

// recordSubstate loads events into the opera, whilst recording substates
func recordSubstate(cfg *utils.Config, log *logging.Logger, mdi *MetadataInfo) error {
	_, err := os.Stat(cfg.Events)
	if os.IsNotExist(err) {
		return fmt.Errorf("supplied events file %s doesn't exist", cfg.Events)
	}

	log.Noticef("Starting Substate recording of %v", cfg.Events)

	cmd := exec.Command("opera", "--datadir", cfg.Db, "--gcmode=full", "--db.preset=legacy-ldb", "--cache", strconv.Itoa(cfg.Cache), "import", "events", "--recording", "--substate-db", cfg.SubstateDb, cfg.Events)

	err = runCommand(cmd, nil, log)
	if err != nil {
		// remove empty substateDb
		return fmt.Errorf("import events; %v", err)
	}

	// retrieve block the opera was iterated into
	cfg.Last, mdi.lastEpoch, err = GetOperaBlock(cfg)

	if err != nil {
		return fmt.Errorf("GetOperaBlock last; %v", err)
	}
	if cfg.First >= cfg.Last {
		return fmt.Errorf("supplied events didn't produce any new blocks")
	}

	log.Noticef("Substates generated for %v - %v", cfg.First, cfg.Last)

	mdi.firstBlock = cfg.First
	mdi.lastBlock = cfg.Last

	err = Merge(cfg, []string{cfg.SubstateDb}, mdi)
	if err != nil {
		return err
	}
	cfg.SubstateDb = cfg.AidaDb

	return nil
}

// initOperaFromGenesis prepares opera by loading genesis
func initOperaFromGenesis(cfg *utils.Config, log *logging.Logger) error {
	cmd := exec.Command("opera", "--datadir", cfg.Db, "--genesis", cfg.Genesis, "--exitwhensynced.epoch=0", "--cache", strconv.Itoa(cfg.Cache), "--db.preset=legacy-ldb", "--maxpeers=0")

	err := runCommand(cmd, nil, log)
	if err != nil {
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

// runCommand wraps cmd execution to distinguish whether to display its output
func runCommand(cmd *exec.Cmd, resultChan chan string, log *logging.Logger) error {
	if resultChan != nil {
		defer close(resultChan)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	defer stderr.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	merged := io.MultiReader(stderr, stdout)
	scanner := bufio.NewScanner(merged)
	if log.IsEnabledFor(logging.DEBUG) {
		for scanner.Scan() {
			m := scanner.Text()
			if resultChan != nil {
				resultChan <- m
			}
			log.Debug(m)
		}
	}
	err = cmd.Wait()

	// command failed
	if err != nil {
		if !log.IsEnabledFor(logging.DEBUG) {
			for scanner.Scan() {
				m := scanner.Text()
				if resultChan != nil {
					resultChan <- m
				}
				log.Error(m)
			}
		}
		return err
	}
	return nil
}

// TODO rewrite after dump is using the config then pass modified cfg directly to the dump function
func prepareDumpCliContext(cfg *utils.Config) (*cli.Context, error) {
	flagSet := flag.NewFlagSet("", 0)
	flagSet.String(utils.WorldStateFlag.Name, cfg.WorldStateDb, "")
	flagSet.String(utils.DbFlag.Name, cfg.Db+"/chaindata/leveldb-fsh/", "")
	flagSet.String(utils.StateDbVariantFlag.Name, "ldb", "")
	flagSet.String(utils.SourceTableNameFlag.Name, utils.SourceTableNameFlag.Value, "")
	flagSet.String(utils.TrieRootHashFlag.Name, utils.TrieRootHashFlag.Value, "")
	flagSet.Int(substate.WorkersFlag.Name, substate.WorkersFlag.Value, "")
	flagSet.Uint64(utils.TargetBlockFlag.Name, utils.TargetBlockFlag.Value, "")
	flagSet.Int(utils.ChainIDFlag.Name, cfg.ChainID, "")
	flagSet.String(logger.LogLevelFlag.Name, cfg.LogLevel, "")

	ctx := cli.NewContext(cli.NewApp(), flagSet, nil)

	err := ctx.Set(utils.DbFlag.Name, cfg.Db+"/chaindata/leveldb-fsh/")
	if err != nil {
		return nil, err
	}
	command := &cli.Command{Name: state.CmdDumpState.Name}
	ctx.Command = command

	return ctx, nil
}

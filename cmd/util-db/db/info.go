package db

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/Fantom-foundation/Aida/cmd/util-db/flags"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/utils/dbcompoment"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/urfave/cli/v2"
)

var InfoCommand = cli.Command{
	Name:  "info",
	Usage: "Prints information about AidaDb",
	Subcommands: []*cli.Command{
		&cmdMetadata,
		&cmdDelAcc,
		&cmdCount,
		&cmdRange,
		&cmdPrintStateHash,
	},
}

var cmdCount = cli.Command{
	Action:    printCount,
	Name:      "count",
	Usage:     "Count records in AidaDb.",
	ArgsUsage: "<firstBlockNum>, <lastBlockNum>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.DbComponentFlag,
		&logger.LogLevelFlag,
	},
}

var cmdRange = cli.Command{
	Action: printRange,
	Name:   "range",
	Usage:  "Prints range of all types in AidaDb",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.DbComponentFlag,
		&logger.LogLevelFlag,
	},
}

var cmdDelAcc = cli.Command{
	Action:    printDeletedAccountInfo,
	Name:      "del-acc",
	Usage:     "Prints info about given deleted account in AidaDb.",
	ArgsUsage: "<firstBlockNum>, <lastBlockNum>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
		&flags.Account,
	},
}

var cmdMetadata = cli.Command{
	Action: printMetadataCmd,
	Name:   "metadata",
	Usage:  "Prints metadata",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
}

var cmdPrintStateHash = cli.Command{
	Action:    printStateHash,
	Name:      "state-hash",
	Usage:     "Prints state hash for given block number.",
	ArgsUsage: "<BlockNum>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
}

func printMetadataCmd(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	return utildb.PrintMetadata(cfg.AidaDb)
}

// printCount prints count of given db component in given AidaDb
func printCount(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if argErr != nil {
		return argErr
	}

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}
	defer utildb.MustCloseDB(aidaDb)

	log := logger.NewLogger(cfg.LogLevel, "AidaDb-Count")
	log.Noticef("Inspecting database between blocks %v-%v", cfg.First, cfg.Last)

	// print substate count
	if *cfg.DbComponent == dbcompoment.Substate || *cfg.DbComponent == dbcompoment.All {
		count := utildb.GetSubstateCount(cfg, aidaDb)
		log.Noticef("Found %v substates", count)
	}

	// print update count
	if *cfg.DbComponent == dbcompoment.Update || *cfg.DbComponent == dbcompoment.All {
		count, err := utildb.GetUpdateCount(cfg, aidaDb)
		if err != nil {
			log.Warningf("cannot print update count; %v", err)
		} else {
			log.Noticef("Found %v updates", count)
		}
	}

	// print deleted count
	if *cfg.DbComponent == dbcompoment.Delete || *cfg.DbComponent == dbcompoment.All {
		count, err := utildb.GetDeletedCount(cfg, aidaDb)
		if err != nil {
			log.Warningf("cannot print deleted count; %v", err)
		} else {
			log.Noticef("Found %v deleted accounts", count)
		}
	}

	// print state hash count
	if *cfg.DbComponent == dbcompoment.StateHash || *cfg.DbComponent == dbcompoment.All {
		count, err := utildb.GetStateHashCount(cfg, aidaDb)
		if err != nil {
			log.Warningf("cannot print state hash count; %v", err)
		} else {
			log.Noticef("Found %v state-hashes", count)
		}
	}

	return nil
}

// printRange prints range of given db component in given AidaDb
func printRange(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	log := logger.NewLogger(cfg.LogLevel, "AidaDb-Range")

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "aidaDb", true)
	if err != nil {
		return fmt.Errorf("error opening aidaDb %s: %v", cfg.AidaDb, err)
	}
	defer utildb.MustCloseDB(aidaDb)

	// print substate range
	if *cfg.DbComponent == dbcompoment.Substate || *cfg.DbComponent == dbcompoment.All {
		substate.SetSubstateDbBackend(aidaDb)
		firstBlock, lastBlock, ok := utils.FindBlockRangeInSubstate()
		if !ok {
			log.Warning("No substate found")
		} else {
			log.Infof("Substate block range: %v - %v", firstBlock, lastBlock)
		}
	}

	// print update range
	if *cfg.DbComponent == dbcompoment.Update || *cfg.DbComponent == dbcompoment.All {
		firstUsBlock, lastUsBlock, err := utildb.FindBlockRangeInUpdate(aidaDb)
		if err != nil {
			log.Warningf("cannot find updateset range; %v", err)
		}
		log.Infof("Updateset block range: %v - %v", firstUsBlock, lastUsBlock)
	}

	// print deleted range
	if *cfg.DbComponent == dbcompoment.Delete || *cfg.DbComponent == dbcompoment.All {
		first, last, err := utildb.FindBlockRangeInDeleted(aidaDb)
		if err != nil {
			log.Warningf("cannot find deleted range; %v", err)
		} else {
			log.Infof("Deleted block range: %v - %v", first, last)
		}
	}

	// print state hash range
	if *cfg.DbComponent == dbcompoment.StateHash || *cfg.DbComponent == dbcompoment.All {
		firstStateHashBlock, lastStateHashBlock, err := utildb.FindBlockRangeInStateHash(aidaDb, log)
		if err != nil {
			log.Warningf("cannot find state hash range; %v", err)
		} else {
			log.Infof("State Hash range: %v - %v", firstStateHashBlock, lastStateHashBlock)
		}
	}
	return nil
}

// printDeletedAccountInfo for given deleted account in AidaDb
func printDeletedAccountInfo(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if argErr != nil {
		return argErr
	}

	log := logger.NewLogger(cfg.LogLevel, "AidaDb-Deleted-Account-Info")

	db, err := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletionDb)
	if err != nil {
		return err
	}

	accounts, err := db.GetAccountsDestroyedInRange(cfg.First, cfg.Last)
	if err != nil {
		return fmt.Errorf("cannot Get all destroyed accounts; %v", err)
	}

	wantedAcc := ctx.String(flags.Account.Name)

	for _, acc := range accounts {
		if strings.Compare(acc.String(), wantedAcc) == 0 {
			log.Noticef("Found record in range %v - %v", cfg.First, cfg.Last)
			return nil
		}

	}

	log.Warningf("Did not find record in range %v - %v", cfg.First, cfg.Last)

	return nil

}

// printTableHash creates hash of substates, updatesets, deletion and state-hashes.
func printTableHash(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("aidaDb %v; %v", cfg.AidaDb, err)
	}

	log := logger.NewLogger(cfg.LogLevel, "printTableHash")
	log.Info("Inspecting database...")
	err = utildb.TableHash(cfg, aidaDb, log)
	if err != nil {
		return err
	}
	log.Info("Finished")

	utildb.MustCloseDB(aidaDb)
	return nil
}

func printStateHash(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.OneToNArgs)
	if argErr != nil {
		return argErr
	}

	log := logger.NewLogger(cfg.LogLevel, "AidaDb-Print-State-Hash")

	blockNum, err := strconv.ParseUint(ctx.Args().Slice()[0], 10, 64)
	if err != nil {
		return err
	}

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}

	hexStr := hexutil.EncodeUint64(blockNum)

	prefix := []byte(utils.StateHashPrefix + hexStr)

	bytes, err := aidaDb.Get(prefix)
	if err != nil {
		return fmt.Errorf("aida-db doesn't contain state hash for block %v", blockNum)
	}

	log.Noticef("State hash for block %v is 0x%v", blockNum, hex.EncodeToString(bytes))

	return nil
}

// printPrefixHash calculates md5 of prefix in given AidaDb
func printPrefixHash(ctx *cli.Context) error {
	log := logger.NewLogger("INFO", "GeneratePrefixHash")

	cfg, err := utils.NewConfig(ctx, utils.NoArgs)

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open db; %v", err)
	}

	defer utildb.MustCloseDB(aidaDb)

	if ctx.Args().Len() == 0 || ctx.Args().Len() >= 2 {
		return fmt.Errorf("generate-prefix-hash command requires exactly 1 argument")
	}

	prefix := ctx.Args().Slice()[0]
	log.Noticef("Generating hash for prefix %v", prefix)
	_, err = utildb.GeneratePrefixHash(aidaDb, prefix, "INFO")
	return err
}

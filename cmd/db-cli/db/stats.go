package db

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Fantom-foundation/Aida/cmd/db-cli/flags"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

var Stats = cli.Command{
	Name:  "stats",
	Usage: "Prints statistics about AidaDb",
	Subcommands: []*cli.Command{
		&cmdPrint,
		&cmdDelAcc,
		&cmdCount,
	},
}

var cmdPrint = cli.Command{
	Action:    printStats,
	Name:      "print",
	Usage:     "Prints metadata of AidaDb",
	ArgsUsage: "<lastBlockNum>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
}

func printStats(ctx *cli.Context) error {
	log := logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "AidaDb-Stats")

	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	// open aidaDb
	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}

	defer MustCloseDB(aidaDb)

	log.Notice("AIDA-DB INFO:")

	if err = printDbType(aidaDb, log); err != nil {
		return err
	}

	// CHAINID
	if err = printChainID(aidaDb, log); err != nil {
		return err
	}

	// BLOCKS
	if err = printBlocks(aidaDb, log); err != nil {
		return err
	}

	// EPOCHS
	if err = printEpochs(aidaDb, log); err != nil {
		return err
	}

	// TIMESTAMP
	if err = printCreateTime(aidaDb, log); err != nil {
		return err
	}

	// UPDATE-SET
	if err = printUpdateSetInfo(aidaDb, log); err != nil {
		return err
	}

	return nil
}

func printBlocks(aidaDb ethdb.Database, log *logging.Logger) error {
	firstBlockBytes, err := aidaDb.Get([]byte(FirstBlockPrefix))
	if err != nil {
		return fmt.Errorf("cannot get first block from db; %v", err)
	}
	log.Infof("First Block: %v", bigendian.BytesToUint64(firstBlockBytes))

	lastBlockBytes, err := aidaDb.Get([]byte(LastBlockPrefix))
	if err != nil {
		return fmt.Errorf("cannot get last block from db; %v", err)
	}
	log.Infof("Last Block: %v", bigendian.BytesToUint64(lastBlockBytes))

	return nil
}

func printEpochs(aidaDb ethdb.Database, log *logging.Logger) error {
	firstEpochBytes, err := aidaDb.Get([]byte(FirstEpochPrefix))
	if err != nil {
		return fmt.Errorf("cannot get first epoch from db; %v", err)
	}
	log.Infof("First Epoch: %v", bigendian.BytesToUint64(firstEpochBytes))

	lastEpochBytes, err := aidaDb.Get([]byte(LastEpochPrefix))
	if err != nil {
		return fmt.Errorf("cannot get last epoch from db; %v", err)
	}
	log.Infof("Last Epoch: %v", bigendian.BytesToUint64(lastEpochBytes))

	return nil
}

func printCreateTime(aidaDb ethdb.Database, log *logging.Logger) error {
	timestampBytes, err := aidaDb.Get([]byte(TimestampPrefix))
	if err != nil {
		return fmt.Errorf("cannot get timestamp from db; %v", err)
	}
	log.Infof("Created: %v", time.Unix(int64(bigendian.BytesToUint64(timestampBytes)), 0))

	return nil
}

func printUpdateSetInfo(aidaDb ethdb.Database, log *logging.Logger) error {
	log.Notice("UPDATE-SET INFO:")

	intervalBytes, err := aidaDb.Get([]byte(substate.UpdatesetIntervalKey))
	if err != nil {
		return fmt.Errorf("cannot get updateset interval from db; %v", err)
	}
	log.Infof("Interval: %v blocks", bigendian.BytesToUint64(intervalBytes))

	sizeBytes, err := aidaDb.Get([]byte(substate.UpdatesetSizeKey))
	if err != nil {
		return fmt.Errorf("cannot get updateset size from db; %v", err)
	}
	u := bigendian.BytesToUint64(sizeBytes)

	// todo convert to mb
	log.Infof("Size: %.1f MB", float64(u)/float64(1_000_000))

	return nil
}

func printDbType(aidaDb ethdb.Database, log *logging.Logger) error {
	typeBytes, err := aidaDb.Get([]byte(TypePrefix))
	if err != nil {
		return errors.New("this aida-b seems to have no metadata")
	}

	var typePrint string
	switch aidaDbType(typeBytes[0]) {
	case genType:
		typePrint = "Generate"
	case cloneType:
		typePrint = "Clone"
	case patchType:
		typePrint = "Patch"
	default:
		typePrint = "Could not decode Db type of key: " + string(typeBytes)
	}

	log.Noticef("DB-Type: %v", typePrint)

	return nil
}

func printChainID(aidaDb ethdb.Database, log *logging.Logger) error {
	chainIDBytes, err := aidaDb.Get([]byte(ChainIDPrefix))
	if err != nil {
		return fmt.Errorf("cannot get chain-id from db; %v", err)
	}

	log.Infof("Chain-ID: %v", bigendian.BytesToUint16(chainIDBytes))
	return nil
}

var cmdCount = cli.Command{
	Name:  "count",
	Usage: "Prints count of records in AidaDb",
	Subcommands: []*cli.Command{
		&cmdCountAll,
		&cmdCountDestroyed,
		&cmdCountSubstate,
	},
}

var cmdCountAll = cli.Command{
	Action: printAllCount,
	Name:   "all",
	Usage:  "List of all records in AidaDb.",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
		&flags.Detailed,
	},
}

var cmdCountDestroyed = cli.Command{
	Action:    printDestroyedCount,
	Name:      "destroyed",
	Usage:     "Prints how many destroyed accounts are in AidaDb between given block range",
	ArgsUsage: "<firstBlockNum>, <lastBlockNum>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
}

var cmdCountSubstate = cli.Command{
	Action:    printSubstateCount,
	Name:      "substate",
	Usage:     "Prints how many substates are in AidaDb between given block range",
	ArgsUsage: "<firstBlockNum>, <lastBlockNum>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
}

func printAllCount(ctx *cli.Context) error {
	log := logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "AidaDb-Stats")

	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	// open aidaDb
	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}

	if ctx.Bool(flags.Detailed.Name) {
		log.Notice("Counting each prefix...")
		logDetailedSize(aidaDb, log)
	} else {
		log.Notice("Counting overall size...")
		log.Noticef("All AidaDb records: %v", getDbSize(aidaDb))
	}

	return nil
}

func logDetailedSize(db ethdb.Database, log *logging.Logger) {
	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	countMap := make(map[string]uint64)

	for iter.Next() {
		countMap[string(iter.Key()[:2])]++
	}

	for key, count := range countMap {
		log.Noticef("Prefix :%v; Count: %v", key, count)
	}
}

func printDestroyedCount(ctx *cli.Context) error {
	log := logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "AidaDb-Stats")

	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if argErr != nil {
		return argErr
	}

	db, err := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletionDb)
	if err != nil {
		return err
	}

	accounts, err := db.GetAccountsDestroyedInRange(cfg.First, cfg.Last)
	if err != nil {
		return fmt.Errorf("cannot get all destroyed accounts; %v", err)
	}

	log.Noticef("Found %v deleted accounts between blocks %v-%v", len(accounts), cfg.First, cfg.Last)

	return nil
}

func printSubstateCount(ctx *cli.Context) error {
	log := logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "AidaDb-Stats")

	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if argErr != nil {
		return argErr
	}

	substate.SetSubstateDb(cfg.AidaDb)
	substate.OpenSubstateDBReadOnly()

	var count uint64

	iter := substate.NewSubstateIterator(cfg.First, 10)
	for iter.Next() {
		if iter.Value().Block > cfg.Last {
			break
		}
		count++
	}

	log.Noticef("Found %v substates between blocks %v-%v", count, cfg.First, cfg.Last)

	return nil
}

var cmdDelAcc = cli.Command{
	Action:    getDelAcc,
	Name:      "del-acc",
	Usage:     "Prints info about given deleted account in AidaDb.",
	ArgsUsage: "<firstBlockNum>, <lastBlockNum>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
		&flags.Account,
	},
}

func getDelAcc(ctx *cli.Context) error {
	log := logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "AidaDb-Stats")

	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if argErr != nil {
		return argErr
	}

	db, err := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletionDb)
	if err != nil {
		return err
	}

	accounts, err := db.GetAccountsDestroyedInRange(cfg.First, cfg.Last)
	if err != nil {
		return fmt.Errorf("cannot get all destroyed accounts; %v", err)
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

// getDbSize retrieves database size
func getDbSize(db ethdb.Database) uint64 {
	var count uint64
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		count++
	}
	return count
}

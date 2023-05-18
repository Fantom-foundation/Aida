package db

import (
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
		&cmdStats,
		&cmdAll,
		&cmdDelAcc,
	},

	Description: `
The stats command requires one argument: <blockNunLast> -- the last block of aida-db.`,
}

var cmdStats = cli.Command{
	Action: printStats,
	Name:   "stats",
	Usage:  "Prints metadata of AidaDb",
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

	log.Notice("Opening AidaDb")
	// open aidaDb
	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}

	firstBlockBytes, err := aidaDb.Get([]byte(FirstBlockPrefix))
	if err != nil {
		return fmt.Errorf("cannot get first block from db; %v", err)
	}

	log.Notice("AIDA-DB INFO:")

	log.Noticef("First Block :%v", bigendian.BytesToUint64(firstBlockBytes))

	lastBlockBytes, err := aidaDb.Get([]byte(LastBlockPrefix))
	if err != nil {
		return fmt.Errorf("cannot get last block from db; %v", err)
	}
	log.Noticef("Last Block: %v", bigendian.BytesToUint64(lastBlockBytes))

	timestampBytes, err := aidaDb.Get([]byte(TimestampPrefix))
	if err != nil {
		return fmt.Errorf("cannot get timestamp from db; %v", err)
	}

	log.Noticef("Created: %v", time.Unix(int64(bigendian.BytesToUint64(timestampBytes)), 0))

	log.Notice("UPDATESET INFO:")

	intervalBytes, err := aidaDb.Get([]byte(substate.UpdatesetIntervalKey))
	if err != nil {
		return fmt.Errorf("cannot get updateset interval from db; %v", err)
	}
	log.Noticef("Interval: %v", bigendian.BytesToUint64(intervalBytes))

	sizeBytes, err := aidaDb.Get([]byte(substate.UpdatesetSizeKey))
	if err != nil {
		return fmt.Errorf("cannot get updateset size from db; %v", err)
	}
	log.Noticef("Interval: %v", bigendian.BytesToUint64(sizeBytes))

	return nil
}

var cmdAll = cli.Command{
	Action: listAllRecords,
	Name:   "all",
	Usage:  "List of all records in AidaDb.",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
		&flags.Detailed,
	},
}

func listAllRecords(ctx *cli.Context) error {
	log := logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "AidaDb-Stats")

	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	log.Notice("Opening AidaDb")
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

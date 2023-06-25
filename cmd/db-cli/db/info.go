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

var InfoCommand = cli.Command{
	Name:  "info",
	Usage: "Prints information about AidaDb",
	Subcommands: []*cli.Command{
		&cmdMetadata,
		&cmdDelAcc,
		&cmdCount,
		&cmdPrintMD5,
	},
}

var cmdCount = cli.Command{
	Name:  "count",
	Usage: "Prints count of records",
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

var cmdPrintMD5 = cli.Command{
	Action: printMD5Sum,
	Name:   "print-md5",
	Usage:  "Creates md5 sum of all data (both key and value) inside AidaDb and prints it",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
}

func printMD5Sum(ctx *cli.Context) error {
	if _, err := validate(ctx.String(utils.AidaDbFlag.Name), "INFO"); err != nil {
		return err
	}

	return nil
}

func printMetadataCmd(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	return printMetadata(cfg.AidaDb)
}

// printMetadata from given AidaDb
func printMetadata(pathToDb string) error {
	// open db
	aidaDb, err := rawdb.NewLevelDBDatabase(pathToDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}

	defer MustCloseDB(aidaDb)

	md := newAidaMetadata(aidaDb, "INFO")

	md.log.Notice("AIDA-DB INFO:")

	if err = printDbType(md); err != nil {
		md.log.Warning("Metadata are not yet in this DB; Looking for block range in substate...")

		fb, lb, err := findBlockRangeInSubstate(pathToDb)
		if err != nil {
			return fmt.Errorf("cannot find block range in substate; %v", err)
		}
		md.log.Noticef("First Block: %v Last Block: %v", fb, lb)
		return nil
	}

	// CHAIN-ID
	chainID, err := md.getChainID()
	if err != nil {
		md.log.Warning("Value for ChainID does not exist in given Dbs metadata")
	} else {
		md.log.Infof("Chain-ID: %v", chainID)
	}

	// BLOCKS
	firstBlock, err := md.getFirstBlock()
	if err != nil {
		md.log.Warning("Value for first block does not exist in given Dbs metadata")
	} else {
		md.log.Infof("First Block: %v", firstBlock)
	}

	lastBlock, err := md.getLastBlock()
	if err != nil {
		md.log.Warning("Value for last block does not exist in given Dbs metadata")
	} else {
		md.log.Infof("Last Block: %v", lastBlock)
	}

	// EPOCHS
	firstEpoch, err := md.getFirstEpoch()
	if err != nil {
		md.log.Warning("Value for first epoch does not exist in given Dbs metadata")
	} else {
		md.log.Infof("First Epoch: %v", firstEpoch)
	}

	lastEpoch, err := md.getLastEpoch()
	if err != nil {
		md.log.Warning("Value for last epoch does not exist in given Dbs metadata")
	} else {
		md.log.Infof("Last Epoch: %v", lastEpoch)
	}

	// TIMESTAMP
	timestamp, err := md.getTimestamp()
	if err != nil {
		md.log.Warning("Value for creation time does not exist in given Dbs metadata")
	} else {
		md.log.Infof("Created: %v", time.Unix(int64(timestamp), 0))
	}

	// UPDATE-SET
	printUpdateSetInfo(md)

	return nil
}

// printUpdateSetInfo from given AidaDb
func printUpdateSetInfo(m *aidaMetadata) {
	m.log.Notice("UPDATE-SET INFO:")

	intervalBytes, err := m.db.Get([]byte(substate.UpdatesetIntervalKey))
	if err != nil {
		m.log.Warning("Value for update-set interval does not exist in given Dbs metadata")
	} else {
		m.log.Infof("Interval: %v blocks", bigendian.BytesToUint64(intervalBytes))
	}

	sizeBytes, err := m.db.Get([]byte(substate.UpdatesetSizeKey))
	if err != nil {
		m.log.Warning("Value for update-set size does not exist in given Dbs metadata")
	} else {
		u := bigendian.BytesToUint64(sizeBytes)

		// todo convert to mb
		m.log.Infof("Size: %.1f MB", float64(u)/float64(1_000_000))
	}
}

// printDbType from given AidaDb
func printDbType(m *aidaMetadata) error {
	t, err := m.getDbType()
	if err != nil {
		return err
	}

	var typePrint string
	switch t {
	case genType:
		typePrint = "Generate"
	case cloneType:
		typePrint = "Clone"
	case patchType:
		typePrint = "Patch"
	default:
		return errors.New("unknown db type")
	}

	m.log.Noticef("DB-Type: %v", typePrint)
	return nil
}

// printAllCount counts all prefixes prints number of occurrences.
// If DetailedFlag is called, then it prints count of each prefix
func printAllCount(ctx *cli.Context) error {
	log := logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "AidaDb-Info")

	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	// open db
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

// logDetailedSize counts and prints all prefix occurrence
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

// printDestroyedCount in given AidaDb
func printDestroyedCount(ctx *cli.Context) error {
	log := logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "AidaDb-InfoCommand")

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

// printSubstateCount in given AidaDb
func printSubstateCount(ctx *cli.Context) error {
	log := logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "AidaDb-InfoCommand")

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

// printDeletedAccountInfo for given deleted account in AidaDb
func printDeletedAccountInfo(ctx *cli.Context) error {
	log := logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "AidaDb-InfoCommand")

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

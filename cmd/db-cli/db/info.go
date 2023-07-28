package db

import (
	"encoding/hex"
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
		&cmdPrintDbHash,
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
		&flags.ForceFlag,
	},
}

var cmdPrintDbHash = cli.Command{
	Action: doDbHash,
	Name:   "db-hash",
	Usage:  "Prints db-hash (md5) inside AidaDb. If this value is not present in metadata it iterates through all of data.",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&flags.InsertFlag,
		&flags.ForceFlag,
	},
}

func doDbHash(ctx *cli.Context) error {
	var (
		insert = ctx.Bool(flags.InsertFlag.Name)
		force  = ctx.Bool(flags.ForceFlag.Name)
	)

	aidaDb, err := rawdb.NewLevelDBDatabase(ctx.String(utils.AidaDbFlag.Name), 1024, 100, "profiling", !insert)
	if err != nil {
		return fmt.Errorf("cannot open db; %v", err)
	}

	defer MustCloseDB(aidaDb)

	var dbHash []byte

	md := utils.NewAidaDbMetadata(aidaDb, "INFO")

	// first try to extract from db
	dbHash = md.GetDbHash()
	if len(dbHash) != 0 && !force {
		fmt.Printf("Db-Hash: %v", hex.EncodeToString(dbHash))

		return nil
	}

	// if not found in db, we need to iterate and create the hash
	if dbHash, err = validate(aidaDb, "INFO"); err != nil {
		return err
	}

	if insert {
		if err = md.SetDbHash(dbHash); err != nil {
			return fmt.Errorf("cannot insert db-hash; %v", err)
		}
	}

	return nil
}

func printMetadataCmd(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	return printMetadata(cfg.AidaDb, ctx.Bool(flags.ForceFlag.Name))
}

// printMetadata from given AidaDb
func printMetadata(pathToDb string, force bool) error {
	log := logger.NewLogger("INFO", "Print-Metadata")
	// open db
	aidaDb, err := rawdb.NewLevelDBDatabase(pathToDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}

	defer MustCloseDB(aidaDb)

	md := utils.NewAidaDbMetadata(aidaDb, "INFO")

	log.Notice("AIDA-DB INFO:")

	if err = printDbType(md); err != nil {
		return err
	}

	var (
		firstBlock, lastBlock uint64
		ok                    bool
	)

	lastBlock = md.GetLastBlock()

	firstBlock = md.GetFirstBlock()

	if (firstBlock == 0 && lastBlock == 0) || force {
		substate.SetSubstateDb(pathToDb)
		substate.OpenSubstateDBReadOnly()
		defer substate.CloseSubstateDB()

		firstBlock, lastBlock, ok = utils.FindBlockRangeInSubstate()
		if !ok {
			return errors.New("no substate found")
		}
	}

	// CHAIN-ID
	chainID := md.GetChainID()

	log.Infof("Chain-ID: %v", chainID)

	// BLOCKS
	log.Infof("First Block: %v", firstBlock)

	log.Infof("Last Block: %v", lastBlock)

	// EPOCHS
	firstEpoch := md.GetFirstEpoch()

	log.Infof("First Epoch: %v", firstEpoch)

	lastEpoch := md.GetLastEpoch()

	log.Infof("Last Epoch: %v", lastEpoch)

	dbHash := md.GetDbHash()

	log.Infof("Db Hash: %v", hex.EncodeToString(dbHash))

	// TIMESTAMP
	timestamp := md.GetTimestamp()

	log.Infof("Created: %v", time.Unix(int64(timestamp), 0))

	// UPDATE-SET
	printUpdateSetInfo(md)

	return nil
}

// printUpdateSetInfo from given AidaDb
func printUpdateSetInfo(m *utils.AidaDbMetadata) {
	log := logger.NewLogger("INFO", "Print-Metadata")

	log.Notice("UPDATE-SET INFO:")

	intervalBytes, err := m.Db.Get([]byte(substate.UpdatesetIntervalKey))
	if err != nil {
		log.Warning("Value for update-set interval does not exist in given Dbs metadata")
	} else {
		log.Infof("Interval: %v blocks", bigendian.BytesToUint64(intervalBytes))
	}

	sizeBytes, err := m.Db.Get([]byte(substate.UpdatesetSizeKey))
	if err != nil {
		log.Warning("Value for update-set size does not exist in given Dbs metadata")
	} else {
		u := bigendian.BytesToUint64(sizeBytes)

		// todo convert to mb
		log.Infof("Size: %.1f MB", float64(u)/float64(1_000_000))
	}
}

// printDbType from given AidaDb
func printDbType(m *utils.AidaDbMetadata) error {
	t := m.GetDbType()

	var typePrint string
	switch t {
	case utils.GenType:
		typePrint = "Generate"
	case utils.CloneType:
		typePrint = "Clone"
	case utils.PatchType:
		typePrint = "Patch"
	case utils.NoType:
		typePrint = "NoType"

	default:
		return errors.New("unknown db type")
	}

	logger.NewLogger("INFO", "Print-Metadata").Noticef("DB-Type: %v", typePrint)

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
		log.Noticef("All AidaDb records: %v", GetDbSize(aidaDb))
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
		return fmt.Errorf("cannot Get all destroyed accounts; %v", err)
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

// GetDbSize retrieves database size
func GetDbSize(db ethdb.Database) uint64 {
	var count uint64
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		count++
	}
	return count
}

package db

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Fantom-foundation/Aida/cmd/util-db/flags"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/syndtr/goleveldb/leveldb"
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
		&cmdPrintDbHash,
		&cmdPrintStateHash,
		&cmdPrintHashesSeparated,
	},
}

var cmdCount = cli.Command{
	Name:  "count",
	Usage: "Prints count of records",
	Subcommands: []*cli.Command{
		&cmdCountAll,
		&cmdCountDeleted,
		&cmdCountSubstate,
		&cmdCountStateHash,
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

var cmdCountDeleted = cli.Command{
	Action:    printDeletedCount,
	Name:      "deleted",
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

var cmdCountStateHash = cli.Command{
	Action:    printStateHashCount,
	Name:      "state-hash",
	Usage:     "Prints how many state-hashes are in AidaDb between given block range",
	ArgsUsage: "<firstBlockNum>, <lastBlockNum>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
}

var cmdRange = cli.Command{
	Name:  "range",
	Usage: "Prints range of type in AidaDb",
	Subcommands: []*cli.Command{
		&cmdSubstateRange,
		&cmdUpdateRange,
		&cmdStateHashRange,
	},
}

var cmdSubstateRange = cli.Command{
	Action: printSubstateRange,
	Name:   "substate",
	Usage:  "Prints range of substate in AidaDb",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
	},
}

var cmdUpdateRange = cli.Command{
	Action: printUpdateRange,
	Name:   "update",
	Usage:  "Prints range of updatesets in AidaDb",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
	},
}

var cmdStateHashRange = cli.Command{
	Action: printStateHashRange,
	Name:   "state-hash",
	Usage:  "Prints range of state-hash in AidaDb",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
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

var cmdPrintDbHash = cli.Command{
	Action: doDbHash,
	Name:   "db-hash",
	Usage:  "Prints db-hash (md5) inside AidaDb. If this value is not present in metadata it iterates through all of data.",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&flags.ForceFlag,
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

var cmdPrintHashesSeparated = cli.Command{
	Action:    generateMd5OfPrefixes,
	Name:      "hashes",
	Usage:     "Prints state hash of db prefixes individually. Or just for single prefix specified in arg.",
	ArgsUsage: "<prefix>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
}

func doDbHash(ctx *cli.Context) error {
	var force = ctx.Bool(flags.ForceFlag.Name)

	aidaDb, err := rawdb.NewLevelDBDatabase(ctx.String(utils.AidaDbFlag.Name), 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open db; %v", err)
	}

	defer utildb.MustCloseDB(aidaDb)

	var dbHash []byte

	log := logger.NewLogger("INFO", "AidaDb-Db-Hash")

	md := utils.NewAidaDbMetadata(aidaDb, "INFO")

	// first try to extract from db
	dbHash = md.GetDbHash()
	if len(dbHash) != 0 && !force {
		log.Infof("Db-Hash: %v", hex.EncodeToString(dbHash))
		return nil
	}

	// if not found in db, we need to iterate and create the hash
	if dbHash, err = utildb.GenerateDbHash(aidaDb, "INFO"); err != nil {
		return err
	}

	fmt.Printf("Db-Hash: %v", hex.EncodeToString(dbHash))
	return nil
}

func printMetadataCmd(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	return utildb.PrintMetadata(cfg.AidaDb)
}

// printAllCount counts all prefixes prints number of occurrences.
// If DetailedFlag is called, then it prints count of each prefix
func printAllCount(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	log := logger.NewLogger(cfg.LogLevel, "AidaDb-All-Count")

	// open db
	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}

	if ctx.Bool(flags.Detailed.Name) {
		log.Notice("Counting each prefix...")
		utildb.LogDetailedSize(aidaDb, log)
	} else {
		log.Notice("Counting overall size...")
		log.Noticef("All AidaDb records: %v", utildb.GetDbSize(aidaDb))
	}

	return nil
}

// printDeletedCount in given AidaDb
func printDeletedCount(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if argErr != nil {
		return argErr
	}

	log := logger.NewLogger(cfg.LogLevel, "AidaDb-Print-Deleted-Count")

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
	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if argErr != nil {
		return argErr
	}

	log := logger.NewLogger(cfg.LogLevel, "AidaDb-Print-Substate-Count")

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

// printStateHashCount in given AidaDb
func printStateHashCount(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if argErr != nil {
		return argErr
	}

	log := logger.NewLogger(cfg.LogLevel, "AidaDb-Print-StateHash-Count")

	var count uint64

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "", true)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}

	hashProvider := utils.MakeStateHashProvider(aidaDb)
	for i := cfg.First; i <= cfg.Last; i++ {
		_, err := hashProvider.GetStateHash(int(i))
		if err != nil {
			if errors.Is(err, leveldb.ErrNotFound) {
				continue
			}
			return err
		}
		count++
	}

	log.Noticef("Found %v state-hashes between blocks %v-%v", count, cfg.First, cfg.Last)

	return nil
}

// printStateHashRange prints state hash range stored in given AidaDb
func printStateHashRange(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	log := logger.NewLogger(cfg.LogLevel, "AidaDb-State-Hash-Range")

	db, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "aida-db", true)
	if err != nil {
		return fmt.Errorf("error opening aida-db %s: %v", cfg.AidaDb, err)
	}

	var firstStateHashBlock, lastStateHashBlock uint64
	firstStateHashBlock, err = utils.GetFirstStateHash(db)
	if err != nil {
		return fmt.Errorf("cannot get first state hash; %v", err)
	}

	lastStateHashBlock, err = utils.GetLastStateHash(db)
	if err != nil {
		log.Infof("Found first state hash at %v", firstStateHashBlock)
		return fmt.Errorf("cannot get last state hash; %v", err)
	}

	log.Infof("State Hash range: %v - %v", firstStateHashBlock, lastStateHashBlock)

	return nil
}

// printSubstateRange prints state substate range stored in given AidaDb
func printSubstateRange(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	log := logger.NewLogger(cfg.LogLevel, "AidaDb-Substate-Range")

	substate.SetSubstateDb(cfg.AidaDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	firstBlock, lastBlock, ok := utils.FindBlockRangeInSubstate()
	if !ok {
		return errors.New("no substate found")
	}

	log.Infof("Substate block range: %v - %v", firstBlock, lastBlock)
	return nil
}

// printUpdateRange prints state updateset range stored in given AidaDb
func printUpdateRange(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	log := logger.NewLogger(cfg.LogLevel, "AidaDb-Update-Range")

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}
	udb := substate.NewUpdateDB(aidaDb)
	defer aidaDb.Close()

	var firstBlock, lastBlock uint64
	// get first updateset - TODO refactor out
	iter := aidaDb.NewIterator([]byte(substate.SubstateAllocPrefix), nil)
	defer iter.Release()

	for iter.Next() {
		firstBlock, err = substate.DecodeUpdateSetKey(iter.Key())
		if err != nil {
			return fmt.Errorf("cannot decode updateset key; %v", err)
		}
		break
	}

	// get last updateset
	lastBlock = udb.GetLastKey()

	log.Infof("Updateset block range: %v - %v", firstBlock, lastBlock)
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

// generateMd5OfPrefixes calculates md5 of all prefixes in given AidaDb separately
func generateMd5OfPrefixes(ctx *cli.Context) error {
	log := logger.NewLogger("INFO", "DbHashGenerateCMD")

	cfg, err := utils.NewConfig(ctx, utils.NoArgs)

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open db; %v", err)
	}

	defer utildb.MustCloseDB(aidaDb)

	if ctx.Args().Len() > 0 {
		prefix := ctx.Args().Slice()[0]
		log.Noticef("Searching for data under prefix %v", prefix)
		return generateMd5For(prefix, cfg, aidaDb, log)
	} else {
		err1 := generateMd5For(substate.Stage1SubstatePrefix, cfg, aidaDb, log)
		err2 := generateMd5For(substate.SubstateAllocPrefix, cfg, aidaDb, log)
		err3 := generateMd5For(substate.DestroyedAccountPrefix, cfg, aidaDb, log)
		err4 := generateMd5For(utils.StateHashPrefix, cfg, aidaDb, log)
		return errors.Join(err1, err2, err3, err4)
	}
}

func generateMd5For(prefix string, cfg *utils.Config, aidaDb ethdb.Database, log logger.Logger) error {
	var err error
	switch prefix {
	case substate.Stage1SubstatePrefix:
		log.Noticef("Starting DbHash generation for %v; this may take several hours...", cfg.AidaDb)
		log.Noticef("Substates...")
		_, err = utildb.GeneratePrefixHash(aidaDb, substate.Stage1SubstatePrefix, "INFO")
		if err != nil {
			return err
		}
	case substate.SubstateAllocPrefix:
		log.Noticef("Updateset...")
		_, err = utildb.GeneratePrefixHash(aidaDb, substate.SubstateAllocPrefix, "INFO")
		if err != nil {
			return err
		}
	case substate.DestroyedAccountPrefix:
		log.Noticef("Deleted...")
		_, err = utildb.GeneratePrefixHash(aidaDb, substate.DestroyedAccountPrefix, "INFO")
		if err != nil {
			return err
		}
	case utils.StateHashPrefix:
		log.Noticef("StateHash...")
		_, err = utildb.GeneratePrefixHash(aidaDb, utils.StateHashPrefix, "INFO")
		if err != nil {
			return err
		}
	}
	return nil
}

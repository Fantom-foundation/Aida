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
	"github.com/Fantom-foundation/Aida/utils/dbcompoment"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
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
		&cmdSignature,
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

// SignatureCmd calculates md5 of actual data stored
var cmdSignature = cli.Command{
	Action: signature,
	Name:   "signature",
	Usage:  "Calculates md5 of decoded objects stored in AidaDb. Using []byte value from database, it decodes it and calculates md5 of the decoded objects.",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.DbComponentFlag,
		&logger.LogLevelFlag,
	},
	Description: `
Creates signatures of substates, updatesets, deletion and state-hashes using decoded objects from database rather than []byte value representation, because that is not deterministic.
`,
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

// signature creates signatures of substates, updatesets, deletion and state-hashes.
func signature(ctx *cli.Context) error {
	// process arguments and flags
	if ctx.Args().Len() != 1 {
		return fmt.Errorf("signature command requires exactly 1 arguments")
	}
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("aidaDb %v; %v", cfg.AidaDb, err)
	}

	log := logger.NewLogger(cfg.LogLevel, "signature")
	log.Info("Inspecting database...")
	err = utildb.DbSignature(cfg, aidaDb, log)
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

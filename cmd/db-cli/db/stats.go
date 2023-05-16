package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/cmd/db-cli/flags"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/urfave/cli/v2"
)

var Stats = cli.Command{
	Name:  "stats",
	Usage: "print number of items in aida-db",
	Subcommands: []*cli.Command{
		&cmdAll,
		&cmdDelAcc,
	},
	Description: `
The stats command requires one argument: <blockNunLast> -- the last block of aida-db.`,
}

var cmdAll = cli.Command{
	Action:      listAllRecords,
	Name:        "all",
	Usage:       "Lists unknown account storages from the world state database.",
	Description: "Command scans for storage keys in the world state database and shows those not available in the address map.",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
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
		return fmt.Errorf("cannot create aida-db; %v", err)
	}

	log.Notice("Counting...")
	log.Noticef("All AidaDb records: %v", getDbSize(aidaDb))
	return nil
}

var cmdDelAcc = cli.Command{
	Action:      getDelAcc,
	Name:        "del-acc",
	Usage:       "Lists unknown account storages from the world state database.",
	Description: "Command scans for storage keys in the world state database and shows those not available in the address map.",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
	},
}

func getDelAcc(ctx *cli.Context) error {
	log := logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "AidaDb-Stats")

	wantedAcc := ctx.String(ctx.String(flags.Account.Name))

	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if argErr != nil {
		return argErr
	}

	db := substate.OpenDestroyedAccountDBReadOnly(cfg.DeletionDb)

	accounts, err := db.GetAccountsDestroyedInRange(cfg.First, cfg.Last)
	if err != nil {
		return fmt.Errorf("cannot get all destroyed accounts; %v", err)
	}

	for _, acc := range accounts {
		if acc.Hash().String() == wantedAcc {
			log.Noticef("Found record in range %v - %v", cfg.First, cfg.Last)
			return nil
		}

	}

	log.Warning("Did not find record in range %v - %v", cfg.First, cfg.Last)

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

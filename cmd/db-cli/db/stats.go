package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/urfave/cli/v2"
)

var DbStats = cli.Command{
	Action: reportAidaDbStats,
	Name:   "db-stats",
	Usage:  "print number of items in aida-db",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
	},

	Description: `
The stats command requires one argument: <blockNunLast> -- the last block of aida-db.`,
}

func reportAidaDbStats(ctx *cli.Context) error {
	log := logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "AidaDb-Stats")
	// process arguments and flags
	if ctx.Args().Len() != 1 {
		return fmt.Errorf("stats command requires exactly 1 arguments")
	}
	cfg, argErr := utils.NewConfig(ctx, utils.LastBlockArg)
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

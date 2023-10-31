package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/urfave/cli/v2"
)

var ScrapeCommand = cli.Command{
	Action: scrapePrepare,
	Name:   "scrape",
	Usage:  "Stores state hashes into AidaDb for given range",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&logger.LogLevelFlag,
	},
}

// scrapePrepare stores state hashes into AidaDb for given range
func scrapePrepare(ctx *cli.Context) error {
	log := logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "AidaDb-Scrape")

	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if argErr != nil {
		return argErr
	}

	log.Infof("Scraping for range %d-%d", cfg.First, cfg.Last)

	db, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "state-hash", false)
	if err != nil {
		return fmt.Errorf("error opening stateHash leveldb %s: %v", cfg.AidaDb, err)
	}
	defer db.Close()

	err = utils.StateHashScraper(cfg.ChainID, db, cfg.First, cfg.Last, log)
	if err != nil {
		return err
	}

	log.Infof("Scraping finished")
	return nil
}

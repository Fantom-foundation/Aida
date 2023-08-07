package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/urfave/cli/v2"
)

// CompactCommand compact given database
var CompactCommand = cli.Command{
	Action: compact,
	Name:   "compact",
	Usage:  "compact target db",
	Flags: []cli.Flag{
		&utils.TargetDbFlag,
	},
	Description: `
Compacts target database.
`,
}

// compact compacts database
func compact(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	log := logger.NewLogger(cfg.LogLevel, "aida-db-compact")

	targetDb, err := rawdb.NewLevelDBDatabase(cfg.TargetDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open db; %v", err)
	}

	log.Notice("Starting compaction")

	err = targetDb.Compact(nil, nil)
	if err != nil {
		return err
	}

	log.Notice("Compaction finished")

	return nil
}

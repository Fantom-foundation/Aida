package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// CompareCommand compares aida-db to target-db whether they are the same
var CompareCommand = cli.Command{
	Action: compareDb,
	Name:   "compare",
	Usage:  "compares aida-db to target-db whether they are the same",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.TargetDbFlag,
		&logger.LogLevelFlag,
	},
	Description: `
Creates clone db is used to create subset of aida-db to have more compact database, but still fully usable for desired block range.
`,
}

// compareDb compares aida-db to target-db whether they are the same
func compareDb(ctx *cli.Context) error {
	// process arguments and flags
	if ctx.Args().Len() != 1 {
		return fmt.Errorf("compare command requires exactly 1 arguments")
	}
	cfg, err := utils.NewConfig(ctx, utils.LastBlockArg)
	if err != nil {
		return err
	}

	aidaDb, targetDb, err := utildb.OpenTwoDatabases(cfg.AidaDb, cfg.TargetDb)
	if err != nil {
		return err
	}

	log := logger.NewLogger(cfg.LogLevel, "compare")

	log.Info("Comparing databases...")
	err = utildb.CompareDatabases(cfg, aidaDb, targetDb, log)
	if err != nil {
		return err
	}

	log.Info("Databases are the same")

	utildb.MustCloseDB(aidaDb)
	utildb.MustCloseDB(targetDb)

	return nil
}

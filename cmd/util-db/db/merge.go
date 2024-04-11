package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/cmd/util-db/flags"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/urfave/cli/v2"
)

// MergeCommand merges given databases into aida-db
var MergeCommand = cli.Command{
	Action: merge,
	Name:   "merge",
	Usage:  "merge source databases into aida-db",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.DeleteSourceDbsFlag,
		&logger.LogLevelFlag,
		&utils.CompactDbFlag,
		&flags.SkipMetadata,
	},
	Description: `
Creates target aida-db by merging source databases from arguments:
<db1> [<db2> <db3> ...]
`,
}

// merge two or more Dbs together
func merge(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.OneToNArgs)
	if err != nil {
		return err
	}

	sourcePaths := make([]string, ctx.Args().Len())
	for i := 0; i < ctx.Args().Len(); i++ {
		sourcePaths[i] = ctx.Args().Get(i)
	}

	// we need a destination where to save merged aida-db
	if cfg.AidaDb == "" {
		return fmt.Errorf("you need to specify where you want aida-db to save (--aida-db)")
	}

	targetDb, err := db.NewDefaultBaseDB(cfg.AidaDb)
	if err != nil {
		return fmt.Errorf("cannot open db; %v", err)
	}

	var (
		dbs []db.BaseDB
		md  *utils.AidaDbMetadata
	)

	if !cfg.SkipMetadata {
		dbs, err = utildb.OpenSourceDatabases(sourcePaths)
		if err != nil {
			return err
		}
		md, err = utils.ProcessMergeMetadata(cfg, targetDb, dbs, sourcePaths)
		if err != nil {
			return err
		}

		targetDb = md.Db

		for _, db := range dbs {
			utildb.MustCloseDB(db)
		}
	}

	dbs, err = utildb.OpenSourceDatabases(sourcePaths)
	if err != nil {
		return err
	}

	m := utildb.NewMerger(cfg, targetDb, dbs, sourcePaths, md)

	if err = m.Merge(); err != nil {
		return err
	}

	m.CloseSourceDbs()

	return m.FinishMerge()
}

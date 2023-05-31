package db

import (
	"errors"
	"fmt"

	"github.com/Fantom-foundation/Aida/cmd/db-cli/flags"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// MergeCommand merges given databases into aida-db
var merCmd = cli.Command{
	Action: mer,
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

type merger struct {
	cfg           *utils.Config
	log           *logging.Logger
	aidaDb        ethdb.Database
	sourceDbPaths []string
}

func mer(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	sourceDbs := make([]string, ctx.Args().Len())
	for i := 0; i < ctx.Args().Len(); i++ {
		sourceDbs[i] = ctx.Args().Get(i)
	}

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open targetDb; %v", err)
	}

	defer MustCloseDB(aidaDb)

	m := &merger{
		cfg:           cfg,
		log:           logger.NewLogger(cfg.LogLevel, "aida-db-merger"),
		aidaDb:        aidaDb,
		sourceDbPaths: sourceDbs,
	}

	return m.merge()
}

func (m *merger) merge() error {
	return errors.New("not implemented yet")
}

package db

import (
	"errors"
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/db-cli/flags"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/google/martian/log"
	"github.com/op/go-logging"
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

type merger struct {
	cfg           *utils.Config
	log           *logging.Logger
	targetDb      ethdb.Database
	sourceDbs     []ethdb.Database
	sourceDbPaths []string
}

func newMerger(cfg *utils.Config, targetDb ethdb.Database, sourceDbs []ethdb.Database, sourceDbPaths []string) *merger {
	return &merger{
		cfg:           cfg,
		log:           logger.NewLogger(cfg.LogLevel, "aida-db-merger"),
		targetDb:      targetDb,
		sourceDbs:     sourceDbs,
		sourceDbPaths: sourceDbPaths,
	}
}

func merge(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
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

	dbs, err := openSourceDatabases(sourcePaths)
	if err != nil {
		return err
	}

	targetDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open aidaDb; %v", err)
	}

	m := newMerger(cfg, targetDb, dbs, sourcePaths)

	return m.merge()
}

func (m *merger) merge() error {
	var (
		err          error
		written      uint64
		totalWritten uint64
	)

	defer m.closeDbs()

	for i, sourceDb := range m.sourceDbs {

		// copy the sourceDb to the target database
		written, err = copyData(sourceDb, m.targetDb)
		if err != nil {
			return err
		}

		totalWritten += written

		m.log.Noticef("Merging of %v", m.sourceDbPaths[i], i+1, len(m.sourceDbs))
		m.log.Noticef("%v / %v finished", i+1, len(m.sourceDbs))
	}

	if m.cfg.CompactDb {
		m.log.Noticef("Starting compaction")
		err = m.targetDb.Compact(nil, nil)
		if err != nil {
			return err
		}
	}

	// delete source databases
	if m.cfg.DeleteSourceDbs {
		for _, path := range m.sourceDbPaths {
			err = os.RemoveAll(path)
			if err != nil {
				return err
			}
			log.Infof("Deleted: %s\n", path)
		}
	}

	m.log.Notice("Merge finished successfully")

	if !m.cfg.SkipMetadata {
		processMergeMetadata(m.targetDb, m.sourceDbs, m.cfg.LogLevel)
	}

	return errors.New("not implemented yet")
}

func (m *merger) closeDbs() {
	for i, db := range m.sourceDbs {
		if err := db.Close(); err != nil {
			m.log.Warning("cannot close source db (%v); %v", m.sourceDbPaths[i], err)
		}
	}

	if err := m.targetDb.Close(); err != nil {
		m.log.Warning("cannot close targetDb; %v", err)
	}
}

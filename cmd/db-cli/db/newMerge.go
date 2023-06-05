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
var MerCmd = cli.Command{
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
	cfg       *utils.Config
	log       *logging.Logger
	targetDb  ethdb.Database
	sourceDbs []ethdb.Database
	dbPaths   []string
	metadata  *aidaMetadata
}

func mer(ctx *cli.Context) error {
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

	m := &merger{
		cfg:       cfg,
		log:       logger.NewLogger(cfg.LogLevel, "aida-db-merger"),
		targetDb:  targetDb,
		sourceDbs: dbs,
		metadata:  newAidaMetadata(targetDb, mergeType, cfg.LogLevel),
		dbPaths:   sourcePaths,
	}

	defer m.closeDbs()

	return m.merge()
}

func (m *merger) merge() error {
	var (
		err          error
		written      uint64
		totalWritten uint64
	)

	for i, sourceDb := range m.sourceDbs {

		// copy the sourceDb to the target database
		written, err = copyData(sourceDb, m.targetDb)
		if err != nil {
			return err
		}

		totalWritten += written

		m.log.Noticef("Merging of %v", m.dbPaths[i], i+1, len(m.sourceDbs))
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
		for _, path := range m.dbPaths {
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
			m.log.Warning("cannot close source db (%v); %v", m.dbPaths[i], err)
		}
	}

	if err := m.targetDb.Close(); err != nil {
		m.log.Warning("cannot close targetDb; %v", err)
	}
}

// openSourceDatabases opens all databases required for merge
func openSourceDatabases(sourceDbPaths []string) ([]ethdb.Database, error) {
	if len(sourceDbPaths) < 1 {
		return nil, fmt.Errorf("no source database were specified\n")
	}

	var sourceDbs []ethdb.Database
	for i := 0; i < len(sourceDbPaths); i++ {
		path := sourceDbPaths[i]
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("source database %s; doesn't exist\n", path)
		}
		db, err := rawdb.NewLevelDBDatabase(path, 1024, 100, "", true)
		if err != nil {
			return nil, fmt.Errorf("source database %s; error: %v", path, err)
		}
		sourceDbs = append(sourceDbs, db)
	}

	return sourceDbs, nil
}

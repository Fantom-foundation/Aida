package db

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/db-cli/flags"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
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

// newMerger returns new instance of merger
func newMerger(cfg *utils.Config, targetDb ethdb.Database, sourceDbs []ethdb.Database, sourceDbPaths []string) *merger {
	return &merger{
		cfg:           cfg,
		log:           logger.NewLogger(cfg.LogLevel, "aida-db-merger"),
		targetDb:      targetDb,
		sourceDbs:     sourceDbs,
		sourceDbPaths: sourceDbPaths,
	}
}

// merge two or more Dbs together
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
		return fmt.Errorf("cannot open db; %v", err)
	}

	m := newMerger(cfg, targetDb, dbs, sourcePaths)

	defer m.closeDbs()

	if err = m.merge(); err != nil {
		return err
	}

	if err = m.finishMerge(); err != nil {
		return err
	}

	return printMetadata(ctx)
}

// finishMerge compacts targetDb and deletes sourceDbs
func (m *merger) finishMerge() error {
	if m.cfg.CompactDb {
		targetDb, err := rawdb.NewLevelDBDatabase(m.cfg.AidaDb, 1024, 100, "profiling", false)
		if err != nil {
			return fmt.Errorf("cannot open db; %v", err)
		}

		m.log.Noticef("Starting compaction")
		err = targetDb.Compact(nil, nil)
		if err != nil {
			return err
		}

		MustCloseDB(targetDb)
	}

	// delete source databases
	if m.cfg.DeleteSourceDbs {
		for _, path := range m.sourceDbPaths {
			err := os.RemoveAll(path)
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

	return nil
}

// merge one or more sourceDbs into targetDb
func (m *merger) merge() error {
	var (
		err          error
		written      uint64
		totalWritten uint64
	)

	for i, sourceDb := range m.sourceDbs {

		// copy the sourceDb to the target database
		written, err = m.copyData(sourceDb)
		if err != nil {
			return err
		}

		totalWritten += written

		if totalWritten == 0 {
			m.log.Warning("merge did not copy any data")
		}

		m.log.Noticef("Merging of %v", m.sourceDbPaths[i])
	}

	return nil
}

// copyData copies data from iterator into target database
func (m *merger) copyData(sourceDb ethdb.Database) (uint64, error) {
	dbBatchWriter := m.targetDb.NewBatch()

	var written uint64
	iter := sourceDb.NewIterator(nil, nil)

	for iter.Next() {
		// do we have another available item?
		key := iter.Key()

		err := dbBatchWriter.Put(key, iter.Value())
		if err != nil {
			return 0, err
		}
		written++

		// writing data in batches
		if dbBatchWriter.ValueSize() > kvdb.IdealBatchSize {
			err = dbBatchWriter.Write()
			if err != nil {
				return 0, fmt.Errorf("batch-writter cannot write data; %v", err)
			}
			dbBatchWriter.Reset()
		}
	}

	if iter.Error() != nil {
		return 0, fmt.Errorf("iterator retuned error: %v", iter.Error())
	}

	// iteration completed - finish write rest of the pending data
	if dbBatchWriter.ValueSize() > 0 {
		err := dbBatchWriter.Write()
		if err != nil {
			return 0, err
		}
	}
	return written, nil
}

// closeDbs (targetDb and sourceDbs) given to merger
func (m *merger) closeDbs() {
	for i, db := range m.sourceDbs {
		if err := db.Close(); err != nil {
			m.log.Warning("cannot close source db (%v); %v", m.sourceDbPaths[i], err)
		}
	}

	if err := m.targetDb.Close(); err != nil {
		m.log.Warningf("cannot close targetDb; %v", err)
	}
}

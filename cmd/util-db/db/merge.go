package db

import (
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/cmd/util-db/flags"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
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
	md            *utils.AidaDbMetadata
	start         time.Time
}

// newMerger returns new instance of merger
func newMerger(cfg *utils.Config, targetDb ethdb.Database, sourceDbs []ethdb.Database, sourceDbPaths []string, md *utils.AidaDbMetadata) *merger {
	return &merger{
		cfg:           cfg,
		log:           logger.NewLogger(cfg.LogLevel, "aida-db-merger"),
		targetDb:      targetDb,
		sourceDbs:     sourceDbs,
		sourceDbPaths: sourceDbPaths,
		md:            md,
		start:         time.Now(),
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

	targetDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open db; %v", err)
	}

	var (
		dbs []ethdb.Database
		md  *utils.AidaDbMetadata
	)

	if !cfg.SkipMetadata {
		dbs, err = openSourceDatabases(sourcePaths)
		if err != nil {
			return err
		}
		md, err = utils.ProcessMergeMetadata(cfg, targetDb, dbs, sourcePaths)
		if err != nil {
			return err
		}

		targetDb = md.Db

		for _, db := range dbs {
			MustCloseDB(db)
		}
	}

	dbs, err = openSourceDatabases(sourcePaths)
	if err != nil {
		return err
	}

	m := newMerger(cfg, targetDb, dbs, sourcePaths, md)

	if err = m.merge(); err != nil {
		return err
	}

	m.closeSourceDbs()

	return m.finishMerge()
}

// finishMerge compacts targetDb and deletes sourceDbs
func (m *merger) finishMerge() error {
	if !m.cfg.SkipMetadata {
		// merge type db does not have epoch calculations yet
		m.md.Db = m.targetDb
		err := m.md.SetAll()
		if err != nil {
			return err
		}
		MustCloseDB(m.targetDb)

		err = printMetadata(m.cfg.AidaDb)
		if err != nil {
			return err
		}
	}

	// delete source databases
	if m.cfg.DeleteSourceDbs {
		for _, path := range m.sourceDbPaths {
			err := os.RemoveAll(path)
			if err != nil {
				return err
			}
			m.log.Infof("Deleted: %s\n", path)
		}
	}

	elapsed := time.Since(m.start)
	m.log.Noticef("Merge finished successfully! Total elapsed time: %v", elapsed.Round(1*time.Second))

	return nil
}

// merge one or more sourceDbs into targetDb
func (m *merger) merge() error {
	var (
		err     error
		written uint64
		elapsed time.Duration
		start   time.Time
	)

	for i, sourceDb := range m.sourceDbs {
		m.log.Noticef("Merging %v...", m.sourceDbPaths[i])
		start = time.Now()

		// copy the sourceDb to the target database
		written, err = m.copyData(sourceDb)
		if err != nil {
			return err
		}

		if written == 0 {
			m.log.Warningf("merge did not copy any data")
		}

		elapsed = time.Since(start)
		m.log.Noticef("Finished merging of %v! It took: %v", m.sourceDbPaths[i], elapsed.Round(1*time.Second))
		m.log.Noticef("Total elapsed time so far: %v", time.Since(m.start).Round(1*time.Second))
	}

	// compact written data
	if m.cfg.CompactDb {
		start = time.Now()
		m.log.Noticef("Starting compaction...")
		err = m.targetDb.Compact(nil, nil)
		if err != nil {
			return fmt.Errorf("cannot compact targetDb; %v", err)
		}
		elapsed = time.Since(start)
		m.log.Noticef("Compaction finished! Elapsed time %v", elapsed.Round(1*time.Second))
	}

	m.log.Noticef("Merge elapsed time: %v", time.Since(m.start).Round(1*time.Second))
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

// closeSourceDbs (sourceDbs) given to merger
func (m *merger) closeSourceDbs() {
	for i, db := range m.sourceDbs {
		if err := db.Close(); err != nil {
			m.log.Warning("cannot close source db (%v); %v", m.sourceDbPaths[i], err)
		}
	}
}

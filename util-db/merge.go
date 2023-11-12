package util_db

import (
	"fmt"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

type Merger struct {
	cfg           *utils.Config
	log           logger.Logger
	targetDb      ethdb.Database
	sourceDbs     []ethdb.Database
	sourceDbPaths []string
	md            *utils.AidaDbMetadata
	start         time.Time
}

// NewMerger returns new instance of Merger
func NewMerger(cfg *utils.Config, targetDb ethdb.Database, sourceDbs []ethdb.Database, sourceDbPaths []string, md *utils.AidaDbMetadata) *Merger {
	return &Merger{
		cfg:           cfg,
		log:           logger.NewLogger(cfg.LogLevel, "aida-db-Merger"),
		targetDb:      targetDb,
		sourceDbs:     sourceDbs,
		sourceDbPaths: sourceDbPaths,
		md:            md,
		start:         time.Now(),
	}
}

// FinishMerge compacts targetDb and deletes sourceDbs
func (m *Merger) FinishMerge() error {
	if !m.cfg.SkipMetadata {
		// merge type db does not have epoch calculations yet
		m.md.Db = m.targetDb
		err := m.md.SetAll()
		if err != nil {
			return err
		}
		MustCloseDB(m.targetDb)

		err = PrintMetadata(m.cfg.AidaDb)
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

// Merge one or more sourceDbs into targetDb
func (m *Merger) Merge() error {
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
func (m *Merger) copyData(sourceDb ethdb.Database) (uint64, error) {
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

// CloseSourceDbs (sourceDbs) given to Merger
func (m *Merger) CloseSourceDbs() {
	for i, db := range m.sourceDbs {
		if err := db.Close(); err != nil {
			m.log.Warning("cannot close source db (%v); %v", m.sourceDbPaths[i], err)
		}
	}
}

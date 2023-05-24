package db

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
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
	},
	Description: `
Creates target aida-db by merging source databases from arguments:
<db1> [<db2> <db3> ...]
`,
}

type aidaDbType byte

const (
	genType aidaDbType = iota
	patchType
	cloneType
	mergeType
)

// merge implements merging command for combining all source data databases into single database used for profiling.
func merge(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	sourceDbs := make([]string, ctx.Args().Len())
	for i := 0; i < ctx.Args().Len(); i++ {
		sourceDbs[i] = ctx.Args().Get(i)
	}

	// open targetDb
	targetDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open targetDb. Error: %v", err)
	}

	defer MustCloseDB(targetDb)

	// when merging, we must find metadataInfo in the dbs we are merging
	return Merge(cfg, sourceDbs, targetDb, &MetadataInfo{dbType: mergeType})
}

// Merge implements merging command for combining all source data databases into single database used for profiling.
func Merge(cfg *utils.Config, sourceDbPaths []string, targetDb ethdb.Database, mdi *MetadataInfo) error {
	log := logger.NewLogger(cfg.LogLevel, "DB Merger")

	// we need a destination where to save merged aida-db
	if cfg.AidaDb == "" {
		return fmt.Errorf("you need to specify where you want aida-db to save (--aida-db)")
	}

	sourceDBs, err := openSourceDatabases(sourceDbPaths)
	if err != nil {
		return err
	}

	// start with putting metadata into new targetDb
	if err = processMetadata(sourceDBs, targetDb, mdi); err != nil {
		return fmt.Errorf("cannot process metadata; %v", err)
	}

	var totalWritten uint64
	for i, sourceDB := range sourceDBs {
		// copy the sourceDB to the target database
		var written uint64
		written, err = copyData(sourceDB, targetDb)
		if err != nil {
			return err
		}
		totalWritten += written
		log.Noticef("Merging of %s finished", sourceDbPaths[i])
		// close finished sourceDB
		MustCloseDB(sourceDB)
	}

	if cfg.CompactDb {
		log.Noticef("Starting compaction")
		err = targetDb.Compact(nil, nil)
		if err != nil {
			return err
		}
	}

	// close target database
	MustCloseDB(targetDb)

	// delete source databases
	if cfg.DeleteSourceDbs {
		for _, path := range sourceDbPaths {
			err = os.RemoveAll(path)
			if err != nil {
				return err
			}
			log.Infof("Deleted: %s\n", path)
		}
	}
	log.Notice("Merge finished successfully")

	return err
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

// copyData copies data from iterator into target database
func copyData(sourceDb ethdb.Database, targetDb ethdb.Database) (uint64, error) {
	dbBatchWriter := targetDb.NewBatch()

	var written uint64
	iter := sourceDb.NewIterator(nil, nil)
	for {
		// do we have another available item?
		if !iter.Next() {
			// iteration completed - finish write rest of the pending data
			if dbBatchWriter.ValueSize() > 0 {
				err := dbBatchWriter.Write()
				if err != nil {
					return 0, err
				}
			}
			return written, nil
		}
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
				return 0, err
			}
			dbBatchWriter.Reset()
		}
	}
}

// MustCloseDB close database safely
func MustCloseDB(db ethdb.Database) {
	if db != nil {
		err := db.Close()
		if err != nil {
			if err.Error() != "leveldb: closed" {
				fmt.Printf("could not close database; %s\n", err.Error())
			}
		}
	}
}

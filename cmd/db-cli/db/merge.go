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
	return Merge(cfg, sourceDbs)
}

// Merge implements merging command for combining all source data databases into single database used for profiling.
func Merge(cfg *utils.Config, sourceDbs []string) error {
	log := logger.NewLogger(cfg.LogLevel, "DB Merger")

	targetDB, sourceDBs, sourceDBPaths, err := openDatabases(cfg.AidaDb, sourceDbs)
	if err != nil {
		return err
	}

	for i, sourceDB := range sourceDBs {
		// copy the sourceDB to the target database
		err = copyData(sourceDB, targetDB)
		if err != nil {
			return err
		}
		log.Noticef("Merging of %s finished", sourceDBPaths[i])
		// close finished sourceDB
		MustCloseDB(sourceDB)
	}

	if cfg.CompactDb {
		log.Noticef("Starting compaction")
		err = targetDB.Compact(nil, nil)
		if err != nil {
			return err
		}
	}

	// close target database
	MustCloseDB(targetDB)

	// delete source databases
	if cfg.DeleteSourceDbs {
		for _, path := range sourceDBPaths {
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

// openDatabases opens all databases required for merge
func openDatabases(targetPath string, sourceDbs []string) (ethdb.Database, []ethdb.Database, []string, error) {
	if len(sourceDbs) < 1 {
		return nil, nil, nil, fmt.Errorf("no source database were specified\n")
	}

	var sourceDBs []ethdb.Database
	var sourceDBPaths []string
	for i := 0; i < len(sourceDbs); i++ {
		path := sourceDbs[i]
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return nil, nil, nil, fmt.Errorf("source database %s; doesn't exist\n", path)
		}
		db, err := rawdb.NewLevelDBDatabase(path, 1024, 100, "", true)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("source database %s; error: %v", path, err)
		}
		sourceDBPaths = append(sourceDBPaths, path)
		sourceDBs = append(sourceDBs, db)
	}

	// open targetDB
	targetDB, err := rawdb.NewLevelDBDatabase(targetPath, 1024, 100, "profiling", false)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("targetDB. Error: %v", err)
	}

	return targetDB, sourceDBs, sourceDBPaths, nil
}

// copyData copies data from source to target database, substitute
func copyData(sourceDB ethdb.Database, targetDB ethdb.Database) error {
	dbBatchWriter := targetDB.NewBatch()

	iter := sourceDB.NewIterator(nil, nil)
	for {
		// do we have another available item?
		if !iter.Next() {
			// iteration completed - finish write rest of the pending data
			if dbBatchWriter.ValueSize() > 0 {
				err := dbBatchWriter.Write()
				if err != nil {
					return err
				}
			}
			return nil
		}
		key := iter.Key()

		err := dbBatchWriter.Put(key, iter.Value())
		if err != nil {
			return err
		}

		// writing data in batches
		if dbBatchWriter.ValueSize() > kvdb.IdealBatchSize {
			err = dbBatchWriter.Write()
			if err != nil {
				return err
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

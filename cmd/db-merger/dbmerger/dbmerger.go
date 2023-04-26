package dbmerger

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/urfave/cli/v2"
)

// DbMerger implements merging command for combining all source data databases into single database used for profiling.
func DbMerger(ctx *cli.Context) error {
	targetPath := ctx.Path(utils.AidaDbFlag.Name)
	log := utils.NewLogger(ctx.String(utils.LogLevel.Name), "DB Merger")

	targetDB, sourceDBs, sourceDBPaths, err := openDatabases(targetPath, ctx.Args())
	if err != nil {
		return err
	}

	for i, sourceDB := range sourceDBs {
		// copy the sourceDB to the target database
		err = copyData(sourceDB, targetDB)
		if err != nil {
			return err
		}
		log.Noticef("Data copying from %s finished\n", sourceDBPaths[i])
		MustCloseDB(sourceDB)
	}

	// close databases
	MustCloseDB(targetDB)

	// delete source databases
	if ctx.Bool(utils.DeleteSourceDbsFlag.Name) {
		for _, path := range sourceDBPaths {
			err = os.RemoveAll(path)
			if err != nil {
				return err
			}
			log.Infof("Deleted: %s\n", path)
		}
	}
	log.Notice("Merge finished successfully\n")

	return err
}

// openDatabases opens all databases required for merge
func openDatabases(targetPath string, args cli.Args) (ethdb.Database, []ethdb.Database, []string, error) {
	if args.Len() < 1 {
		return nil, nil, nil, fmt.Errorf("no source database were specified\n")
	}

	var sourceDBs []ethdb.Database
	var sourceDBPaths []string
	for i := 0; i < args.Len(); i++ {
		path := args.Get(i)
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

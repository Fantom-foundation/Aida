package db

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/db-cli/flags"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/urfave/cli/v2"
)

// MergeCommand merges given databases into aida-db
var MergeCommand = cli.Command{
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

type aidaDbType byte

const (
	noType aidaDbType = iota
	genType
	patchType
	cloneType
	mergeType
	updateType
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

	// when merging, we must find metadataInfo in the dbs we are merging
	return Merge(cfg, sourceDbs, &aidaMetadata{dbType: mergeType})
}

// Merge implements merging command for combining all source data databases into single database used for profiling.
func Merge(cfg *utils.Config, sourceDbPaths []string, mdi *aidaMetadata) error {
	log := logger.NewLogger(cfg.LogLevel, "DB Merger")

	// open targetDb
	targetDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open targetDb; %v", err)
	}

	defer MustCloseDB(targetDb)

	chainIdBytes, _ := targetDb.Get([]byte(ChainIDPrefix))
	if chainIdBytes != nil {
		u := bigendian.BytesToUint16(chainIdBytes)
		cfg.ChainID = int(u)
	}

	// we need a destination where to save merged aida-db
	if cfg.AidaDb == "" {
		return fmt.Errorf("you need to specify where you want aida-db to save (--aida-db)")
	}

	sourceDBs, err := openSourceDatabases(sourceDbPaths)
	if err != nil {
		return err
	}

	if !cfg.SkipMetadata {

		if err = findMetadata(sourceDBs, targetDb, mdi); err != nil {
			return fmt.Errorf("cannot find metadata in source dbs; %v", err)
		}

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

	if !cfg.SkipMetadata {
		if err = putMetadata(targetDb, mdi); err != nil {
			return fmt.Errorf("cannot put metadata into new aida-db")
		}
	}

	return err
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

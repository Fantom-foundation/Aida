package dbmerger

import (
	"fmt"
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/urfave/cli/v2"
)

// DbMerger implements merging command for combining all source data databases into single database used for profiling.
func DbMerger(ctx *cli.Context) error {
	targetPath := ctx.Path(utils.DBFlag.Name)
	substatePath := ctx.Path(substate.SubstateDirFlag.Name)
	updatedbPath := ctx.Path(utils.UpdateDBDirFlag.Name)
	deletedAccountsPath := ctx.Path(utils.DeletedAccountDirFlag.Name)

	targetDB, substateDB, updatesetDB, deletedAccountsDB, skipSubstate, err := openDatabases(ctx, targetPath, substatePath, updatedbPath, deletedAccountsPath)
	if err != nil {
		return err
	}

	// if deletion of source data is enabled than substate data was already moved to target from source database path by renaming
	if !skipSubstate {
		// copy the substates to the target database
		err = copyData(substateDB, targetDB)
		if err != nil {
			return err
		}
		log.Printf("substate move finished\n")
	}

	// copy the updateset to the target database
	err = copyData(updatesetDB, targetDB)
	if err != nil {
		return err
	}
	log.Printf("updateset move finished\n")

	// copy the deleted accounts to the target database
	err = copyData(deletedAccountsDB, targetDB)
	if err != nil {
		return err
	}
	log.Printf("deleted accounts move finished\n")

	// close databases
	MustCloseDB(targetDB)
	MustCloseDB(substateDB)
	MustCloseDB(updatesetDB)
	MustCloseDB(deletedAccountsDB)

	// delete
	if ctx.Bool(utils.DeleteSourceDBsFlag.Name) {
		err = os.RemoveAll(substatePath)
		if err != nil {
			return err
		}
		err = os.RemoveAll(updatedbPath)
		if err != nil {
			return err
		}
		err = os.RemoveAll(deletedAccountsPath)
		if err != nil {
			return err
		}
	}

	return err
}

// openDatabases opens all databases required for merge
func openDatabases(ctx *cli.Context, targetPath string, substatePath string, updatedbPath string, deletedAccountsPath string) (ethdb.Database, ethdb.Database, ethdb.Database, ethdb.Database, bool, error) {
	_, err := os.Stat(targetPath)
	if !os.IsNotExist(err) {
		return nil, nil, nil, nil, false, fmt.Errorf("target database %s is not empty\n", targetPath)
	}

	// open substateDB
	substateDB, err := rawdb.NewLevelDBDatabase(substatePath, 1024, 100, "substatedir", true)
	if err != nil {
		return nil, nil, nil, nil, false, fmt.Errorf("substateDB. Error: %v", err)
	}

	// open updatesetDB
	updatesetDB, err := rawdb.NewLevelDBDatabase(updatedbPath, 1024, 100, "updatesetdir", true)
	if err != nil {
		return nil, nil, nil, nil, false, fmt.Errorf("updateSetDB. Error: %v", err)
	}

	// open deletedAccountsDB
	deletedAccountsDB, err := rawdb.NewLevelDBDatabase(deletedAccountsPath, 1024, 100, "destroyed_accounts", true)
	if err != nil {
		return nil, nil, nil, nil, false, fmt.Errorf("deletedAccountsDB. Error: %v", err)
	}

	err = checkCompatibility(substateDB, updatesetDB, deletedAccountsDB)
	if err != nil {
		return nil, nil, nil, nil, false, fmt.Errorf("database source data are not compatible. Error: %v", err)
	}

	var skipSubstate = false

	if ctx.Bool(utils.DeleteSourceDBsFlag.Name) {
		// source db has to be deleted we can move the folder to target
		MustCloseDB(substateDB)
		err = os.Rename(substatePath, targetPath)
		if err == nil {
			log.Print("substate move finished\n")
			skipSubstate = true
		} else {
			return nil, nil, nil, nil, false, err
		}
	}

	// open targetDB
	targetDB, err := rawdb.NewLevelDBDatabase(targetPath, 1024, 100, "profiling", false)
	if err != nil {
		return nil, nil, nil, nil, false, fmt.Errorf("targetDB. Error: %v", err)
	}

	return targetDB, substateDB, updatesetDB, deletedAccountsDB, skipSubstate, nil
}

// checkCompatibility confirms that the given databases are compatible
func checkCompatibility(substateDB ethdb.Database, updatesetDB ethdb.Database, deletedAccountsDB ethdb.Database) error {
	// TODO check block ranges of databases
	return nil
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
				log.Printf("could not close database; %s\n", err.Error())
			}
		}
	}
}

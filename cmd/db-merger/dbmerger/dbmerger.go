package dbmerger

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/urfave/cli/v2"
)

// DbMerger implements merging command for combining all source data databases into single database used for profiling.
func DbMerger(ctx *cli.Context) error {
	targetPath := ctx.Path(utils.ProfileDBFlag.Name)
	substatePath := ctx.Path(substate.SubstateDirFlag.Name)
	updatedbPath := ctx.Path(utils.UpdateDBDirFlag.Name)
	deletedAccountsPath := ctx.Path(utils.DeletedAccountDirFlag.Name)

	targetDB, substateDB, updatesetDB, deletedAccountsDB, skipSubstate, err := openDatabases(ctx, targetPath, substatePath, updatedbPath, deletedAccountsPath)
	if err != nil {
		return err
	}

	defer MustCloseDB(targetDB)

	var substateCount, updatesetCount, deletedAccCount uint64

	if !skipSubstate {
		substateCount, err = copyFrom(substateDB, targetDB, nil)
		if err != nil {
			return err
		}
		log.Printf("substate %d items\n", substateCount)
	}

	// previous updatesetDB used 1c prefix instead of 2c to avoid collision with substate prefix has to be replaced
	updatesetCount, err = copyFrom(updatesetDB, targetDB, map[string]string{"1c": "2c"})
	if err != nil {
		return err
	}
	log.Printf("updateset %d items\n", updatesetCount)

	deletedAccCount, err = copyFrom(deletedAccountsDB, targetDB, nil)
	if err != nil {
		return err
	}
	log.Printf("deleted accounts %d items\n", deletedAccCount)

	count := 0
	prefixes := map[string]uint64{}
	iter := targetDB.NewIterator(nil, nil)

verifyLoop:
	for {
		// do we have another available item?
		if !iter.Next() {
			log.Printf("targetDB %d - %v\n", count, prefixes)
			break verifyLoop
		}
		prefixes[string(iter.Key())[:2]]++
		count++
	}

	// close databases
	MustCloseDB(substateDB)
	MustCloseDB(updatesetDB)
	MustCloseDB(deletedAccountsDB)
	if ctx.Bool(utils.DeleteSourceDBsFlag.Name) {
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
			log.Print("substate moved completely\n")
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
	// TODO
	return nil
}

// copyFrom copies data from source to target database, substitute
func copyFrom(sourceDB ethdb.Database, targetDB ethdb.Database, substituteArr map[string]string) (uint64, error) {
	dbBatchWriter := targetDB.NewBatch()
	var count uint64 = 0

	iter := sourceDB.NewIterator(nil, nil)
	for {
		// do we have another available item?
		if !iter.Next() {
			// iteration completed - finish write of pending data
			if dbBatchWriter.ValueSize() > 0 {
				err := dbBatchWriter.Write()
				if err != nil {
					return count, err
				}
			}
			return count, nil
		}
		key := iter.Key()
		pref := string(key[:2])
		v, ok := substituteArr[pref]
		if ok {
			// fix to correct prefix
			key = []byte(strings.Replace(string(key), pref, v, 1))
		}

		count++
		err := dbBatchWriter.Put(key, iter.Value())
		if err != nil {
			return count, err
		}

		if dbBatchWriter.ValueSize() > kvdb.IdealBatchSize {
			err = dbBatchWriter.Write()
			if err != nil {
				return count, err
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

package utildb

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

func CompareDatabases(aidaDb, targetDb ethdb.Database) error {

	aidaDbSubstateHash, err := GetSubstateHash(aidaDb)
	if err != nil {
		return err
	}
	targetDbSubstateHash, err := GetSubstateHash(targetDb)
	if err != nil {
		return err
	}

	if string(aidaDbSubstateHash) != string(targetDbSubstateHash) {
		return fmt.Errorf("substate hash mismatch aidaDb: %s targetDb: %s", aidaDbSubstateHash, targetDbSubstateHash)
	}

	aidaDbDeletionHash, err := GetDeletionHash(aidaDb)
	if err != nil {
		return err
	}

	targetDbDeletionHash, err := GetDeletionHash(targetDb)
	if err != nil {
		return err
	}

	if string(aidaDbDeletionHash) != string(targetDbDeletionHash) {
		return fmt.Errorf("deletion hash mismatch aidaDb: %s targetDb: %s", aidaDbDeletionHash, targetDbDeletionHash)
	}

	aidaDbUpdateDbHash, err := GetUpdateDbHash(aidaDb)
	if err != nil {
		return err
	}

	targetDbUpdateDbHash, err := GetUpdateDbHash(targetDb)
	if err != nil {
		return err
	}

	if string(aidaDbUpdateDbHash) != string(targetDbUpdateDbHash) {
		return fmt.Errorf("updateDb hash mismatch aidaDb: %s targetDb: %s", aidaDbUpdateDbHash, targetDbUpdateDbHash)
	}

	return nil
}

func GetDeletionHash(db ethdb.Database) ([]byte, error) {
	ddb := substate.NewDestroyedAccountDB(db)
	inRange, err := ddb.GetAccountsDestroyedInRange(0, 9999999999999999999)
	if err != nil {
		return nil, err
	}

	hash := md5.New()
	for _, address := range inRange {
		hash.Write(address.Bytes())
	}

	return hash.Sum(nil), nil
}

func GetUpdateDbHash(db ethdb.Database) ([]byte, error) {
	udb := substate.NewUpdateDB(db)

	iter := substate.NewUpdateSetIterator(udb, 0, 9999999999999999999)
	defer iter.Release()

	hash := md5.New()
	for iter.Next() {
		value := iter.Value()
		jsonData, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		hash.Write(jsonData)
	}
	return hash.Sum(nil), nil
}

func GetSubstateHash(db ethdb.Database) ([]byte, error) {
	substate.SetSubstateDbBackend(db)
	it := substate.NewSubstateIterator(0, 20)
	defer it.Release()

	hash := md5.New()

	for it.Next() {
		value := it.Value()

		//substateRLP := substate.NewSubstateRLP(value.Substate)
		//res, err := rlp.EncodeToBytes(substateRLP)
		//if err != nil {
		//	panic(err)
		//}
		//hash.Write(res)
		//fmt.Printf("hash: %v\n", hash.Sum(nil))
		//hash.Reset()
		jsonData, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		hash.Write(jsonData)
	}
	return hash.Sum(nil), nil
}

// OpenTwoDatabases prepares aida and target databases
func OpenTwoDatabases(aidaDbPath, targetDbPath string) (ethdb.Database, ethdb.Database, error) {
	var err error

	// if source db doesn't exist raise error
	_, err = os.Stat(aidaDbPath)
	if os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("specified aida-db %v is empty\n", aidaDbPath)
	}

	// if target db exists raise error
	_, err = os.Stat(targetDbPath)
	if os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("specified target-db %v already exists\n", targetDbPath)
	}

	var aidaDb, cloneDb ethdb.Database

	// open db
	aidaDb, err = rawdb.NewLevelDBDatabase(aidaDbPath, 1024, 100, "profiling", true)
	if err != nil {
		return nil, nil, fmt.Errorf("aidaDb %v; %v", aidaDbPath, err)
	}

	// open createDbClone
	cloneDb, err = rawdb.NewLevelDBDatabase(targetDbPath, 1024, 100, "profiling", true)
	if err != nil {
		return nil, nil, fmt.Errorf("targetDb %v; %v", targetDbPath, err)
	}

	return aidaDb, cloneDb, nil
}

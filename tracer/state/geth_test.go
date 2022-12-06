package state

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

const N = 1000

func fillDb(t *testing.T, directory string) (common.Hash, error) {
	db, err := MakeGethStateDB(directory, "", false)
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	for i := 0; i < N; i++ {
		address := common.Address{byte(i), byte(i >> 8)}
		db.SetNonce(address, 12)
		key := common.Hash{byte(i >> 8), byte(i)}
		value := common.Hash{byte(15)}
		db.SetState(address, key, value)
	}

	hash := db.IntermediateRoot(true)
	//hash := db.(*gethStateDb).db.(*geth.StateDB).Commit(true)
	if err = db.Close(); err != nil {
		t.Fatalf("Failed to close DB: %v", err)
	}
	return hash, nil
}

func TestGethDbFilling(t *testing.T) {
	dir, err := ioutil.TempDir("", "test_db_*")
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", dir)
	}
	if _, err := fillDb(t, dir); err != nil {
		t.Errorf("Unable to fill DB: %v", err)
	}
}

func TestGethDbReloadData(t *testing.T) {
	dir, err := ioutil.TempDir("", "test_db_*")
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", dir)
	}
	hash, err := fillDb(t, dir)
	if err != nil {
		t.Errorf("Unable to fill DB: %v", err)
	}

	// Re-open the data base.
	db, err := OpenGethStateDB(dir, hash, false)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	for i := 0; i < N; i++ {
		address := common.Address{byte(i), byte(i >> 8)}
		if got := db.GetNonce(address); got != 12 {
			t.Fatalf("Nonce of %v is not 12: %v", address, got)
		}
		key := common.Hash{byte(i >> 8), byte(i)}
		value := common.Hash{byte(15)}
		if got := db.GetState(address, key); got != value {
			t.Fatalf("Value of %v/%v is not %v: %v", address, key, value, got)
		}
	}
	if err = db.Close(); err != nil {
		t.Fatalf("Failed to close DB: %v", err)
	}
}

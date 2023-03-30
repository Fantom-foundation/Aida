package state

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

const N = 1000

func fillDb(t *testing.T, directory string) (common.Hash, error) {
	db, err := MakeGethStateDB(directory, "", common.Hash{}, false)
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

	hash, err := db.Commit(true)
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
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
	db, err := MakeGethStateDB(dir, "", hash, false)
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

func TestGethDb_CodeUpdateRetention(t *testing.T) {
	dir, err := ioutil.TempDir("", "test_db_*")
	defer os.RemoveAll(dir)

	db, err := MakeGethStateDB(dir, "", common.Hash{}, false)
	defer func() {
		if err = db.Close(); err != nil {
			t.Fatalf("Failed to close DB: %v", err)
		}
	}()

	addr1 := common.Address{1}
	code0 := []byte{}
	code1 := []byte{1, 2, 3}

	// --- Block 1 ---

	db.BeginBlock(1)

	// Here the account is created and a code is assigne.d
	db.BeginTransaction(0)
	db.CreateAccount(addr1)
	db.SetCode(addr1, code1)
	db.EndTransaction()

	// In this transaction it is confirmed that the code is available.
	db.BeginTransaction(1)
	if got := db.GetCode(addr1); !bytes.Equal(code1, got) {
		t.Errorf("incorrect code, expected %v, got %v", code1, got)
	}
	db.EndTransaction()

	db.EndBlock()

	// --- Block 2 ---

	db.BeginBlock(2)

	// In this transaction we check that the code is still correct and
	// we delete the account, after which the code should be gone (=code0)
	db.BeginTransaction(0)
	if got := db.GetCode(addr1); !bytes.Equal(code1, got) {
		t.Errorf("incorrect code, expected %v, got %v", code1, got)
	}
	// db.Suicide(addr1)  // < when adding this, the test passes
	db.CreateAccount(addr1)
	if got := db.GetCode(addr1); !bytes.Equal(code0, got) {
		t.Errorf("incorrect code, expected %v, got %v", code1, got)
	}
	db.EndTransaction()

	// Before ending the block, we check that the code is indeed gone.
	db.BeginTransaction(0)
	if got := db.GetCode(addr1); !bytes.Equal(code0, got) {
		t.Errorf("incorrect code, expected %v, got %v", code0, got)
	}
	db.EndTransaction()

	db.EndBlock()

	// --- Block 3 ---

	db.BeginBlock(3)

	// In the sub-sequent block, the code should be still gone.
	db.BeginTransaction(0)
	if got := db.GetCode(addr1); !bytes.Equal(code0, got) {
		t.Errorf("incorrect code, expected %v, got %v", code0, got)
	}
	db.EndTransaction()
	db.EndBlock()
}

func TestGethDb_NonceUpdateRetention(t *testing.T) {
	dir, err := ioutil.TempDir("", "test_db_*")
	defer os.RemoveAll(dir)

	db, err := MakeGethStateDB(dir, "", common.Hash{}, false)
	defer func() {
		if err = db.Close(); err != nil {
			t.Fatalf("Failed to close DB: %v", err)
		}
	}()

	addr1 := common.Address{1}

	// --- Block 1 ---

	db.BeginBlock(1)

	db.BeginTransaction(0)
	db.CreateAccount(addr1)
	db.SetNonce(addr1, 12) // the nonce is 12
	db.EndTransaction()

	db.EndBlock()

	// --- Block 2 ---

	db.BeginBlock(2)
	db.BeginTransaction(0)
	if got := db.GetNonce(addr1); got != 12 {
		t.Errorf("incorrect nonce, expected %v, got %v", 12, got)
	}
	db.CreateAccount(addr1) // the nonce is reverted to 0
	if got := db.GetNonce(addr1); got != 0 {
		t.Errorf("incorrect nonce, expected %v, got %v", 0, got)
	}
	db.EndTransaction()
	db.EndBlock()

	// --- Block 3 ---

	db.BeginBlock(3)

	db.BeginTransaction(0)
	// Here, the nonce should be 0, as it was reported at the end of the predecessor block.
	if got := db.GetNonce(addr1); got != 0 {
		t.Errorf("incorrect nonce, expected %v, got %v", 0, got)
	}
	db.EndTransaction()
	db.EndBlock()
}

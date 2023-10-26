package utils

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
)

// TestStateHash_ZeroHasSameStateHashAsOne tests that the state hash of block 0 is the same as the state hash of block 1
func TestStateHash_ZeroHasSameStateHashAsOne(t *testing.T) {
	tmpDir := t.TempDir() + "/blockHashes"
	err := StateHashScraper(4002, tmpDir, 0, 1)
	if err != nil {
		t.Fatalf("error scraping state hashes: %v", err)
	}
	db, err := rawdb.NewLevelDBDatabase(tmpDir, 1024, 100, "state-hash", true)
	if err != nil {
		t.Fatalf("error opening stateHash leveldb %s: %v", tmpDir, err)
	}
	defer db.Close()

	shp := MakeStateHashProvider(db)

	hashZero, err := shp.GetStateHash(0)
	if err != nil {
		t.Fatalf("error getting state hash for block 0: %v", err)
	}

	hashOne, err := shp.GetStateHash(100)
	if err != nil {
		t.Fatalf("error getting state hash for block 100: %v", err)
	}

	if hashZero != hashOne {
		t.Fatalf("state hash of block 0 (%s) is not the same as the state hash of block 1 (%s)", hashZero.Hex(), hashOne.Hex())
	}
}

// TestStateHash_ZeroHasSameStateHashAsOne tests that the state hash of block 0 is the same as the state hash of block 1
func TestStateHash_ZeroHasDifferentStateHashAfterHundredBlocks(t *testing.T) {
	tmpDir := t.TempDir() + "/blockHashes"
	err := StateHashScraper(4002, tmpDir, 0, 100)
	if err != nil {
		t.Fatalf("error scraping state hashes: %v", err)
	}
	db, err := rawdb.NewLevelDBDatabase(tmpDir, 1024, 100, "state-hash", true)
	if err != nil {
		t.Fatalf("error opening stateHash leveldb %s: %v", tmpDir, err)
	}
	defer db.Close()

	shp := MakeStateHashProvider(db)

	hashZero, err := shp.GetStateHash(0)
	if err != nil {
		t.Fatalf("error getting state hash for block 0: %v", err)
	}

	hashHundred, err := shp.GetStateHash(100)
	if err != nil {
		t.Fatalf("error getting state hash for block 100: %v", err)
	}

	// block 0 should have a different state hash than block 100
	if hashZero == hashHundred {
		t.Fatalf("state hash of block 0 (%s) is the same as the state hash of block 100 (%s)", hashZero.Hex(), hashHundred.Hex())
	}
}

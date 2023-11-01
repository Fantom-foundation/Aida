package utils

import (
	"context"
	"testing"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/rpc"
)

// TestStateHash_ZeroHasSameStateHashAsOne tests that the state hash of block 0 is the same as the state hash of block 1
func TestStateHash_ZeroHasSameStateHashAsOne(t *testing.T) {
	tmpDir := t.TempDir() + "/blockHashes"
	db, err := rawdb.NewLevelDBDatabase(tmpDir, 1024, 100, "state-hash", false)
	if err != nil {
		t.Fatalf("error opening stateHash leveldb %s: %v", tmpDir, err)
	}
	log := logger.NewLogger("info", "Test state hash")

	err = StateHashScraper(nil, 4002, "", db, 0, 1, log)
	if err != nil {
		t.Fatalf("error scraping state hashes: %v", err)
	}
	err = db.Close()
	if err != nil {
		t.Fatalf("error closing stateHash leveldb %s: %v", tmpDir, err)
	}

	db, err = rawdb.NewLevelDBDatabase(tmpDir, 1024, 100, "state-hash", true)
	if err != nil {
		t.Fatalf("error opening stateHash leveldb %s: %v", tmpDir, err)
	}
	defer db.Close()

	shp := MakeStateHashProvider(db)

	hashZero, err := shp.GetStateHash(0)
	if err != nil {
		t.Fatalf("error getting state hash for block 0: %v", err)
	}

	hashOne, err := shp.GetStateHash(1)
	if err != nil {
		t.Fatalf("error getting state hash for block 1: %v", err)
	}

	if hashZero != hashOne {
		t.Fatalf("state hash of block 0 (%s) is not the same as the state hash of block 1 (%s)", hashZero.Hex(), hashOne.Hex())
	}
}

// TestStateHash_ZeroHasSameStateHashAsOne tests that the state hash of block 0 is different to the state hash of block 100
// we are expecting that at least some storage has changed between block 0 and block 100
func TestStateHash_ZeroHasDifferentStateHashAfterHundredBlocks(t *testing.T) {
	tmpDir := t.TempDir() + "/blockHashes"
	db, err := rawdb.NewLevelDBDatabase(tmpDir, 1024, 100, "state-hash", false)
	if err != nil {
		t.Fatalf("error opening stateHash leveldb %s: %v", tmpDir, err)
	}
	log := logger.NewLogger("info", "Test state hash")

	err = StateHashScraper(nil, 4002, "", db, 0, 100, log)
	if err != nil {
		t.Fatalf("error scraping state hashes: %v", err)
	}
	err = db.Close()
	if err != nil {
		t.Fatalf("error closing stateHash leveldb %s: %v", tmpDir, err)
	}

	db, err = rawdb.NewLevelDBDatabase(tmpDir, 1024, 100, "state-hash", true)
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

func TestStateHash_KeyToUint64(t *testing.T) {
	type args struct {
		hexBytes []byte
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr bool
	}{
		{"testZeroConvert", args{[]byte(StateHashPrefix + "0x0")}, 0, false},
		{"testOneConvert", args{[]byte(StateHashPrefix + "0x1")}, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StateHashKeyToUint64(tt.args.hexBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("StateHashKeyToUint64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("StateHashKeyToUint64() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getClient(t *testing.T) {
	type args struct {
		ctx     context.Context
		chainId ChainID
		ipcPath string
	}
	log := logger.NewLogger("info", "Test state hash")
	tests := []struct {
		name    string
		args    args
		want    *rpc.Client
		wantErr bool
	}{
		{"testGetClientRpcMainnet", args{context.Background(), 250, ""}, &rpc.Client{}, false},
		{"testGetClientRpcTestnet", args{context.Background(), 4002, ""}, &rpc.Client{}, false},
		{"testGetClientIpcNonExistant", args{context.Background(), 4002, "/non-existant-path"}, nil, false},
		{"testGetClientRpcUnknownChainId", args{context.Background(), 88888, ""}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getClient(tt.args.ctx, tt.args.chainId, tt.args.ipcPath, log)
			if (err != nil) != tt.wantErr {
				t.Errorf("getClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != nil && got == nil {
				t.Errorf("getClient() got nil, want non-nil")
			}
		})
	}
}

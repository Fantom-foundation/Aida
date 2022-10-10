// Package db implements database interfaces for the world state manager.
package db

import (
	"fmt"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"io"
	"log"
)

const (
	// CodePrefix represents a prefix added to the code hash to separate code and state data in the KV database.
	// CodePrefix + codeHash (256-bit) -> code
	CodePrefix = "1c"
)

var (
	BlockNumberKey = []byte("blockNumberKey")
)

// StateSnapshotDB represents the state snapshot database handle.
type StateSnapshotDB struct {
	Backend BackendDatabase
}

// BackendDatabase represents the underlying KV store used for the StateSnapshotDB
type BackendDatabase interface {
	ethdb.KeyValueReader
	ethdb.KeyValueWriter
	ethdb.Batcher
	ethdb.Iteratee
	ethdb.Stater
	ethdb.Compacter
	io.Closer
}

// OpenStateSnapshotDB opens state snapshot database at the given path.
func OpenStateSnapshotDB(path string) (*StateSnapshotDB, error) {
	backend, err := rawdb.NewLevelDBDatabase(path, 1024, 100, "substatedir", false)
	if err != nil {
		return nil, err
	}

	return &StateSnapshotDB{Backend: backend}, nil
}

// MustCloseSnapshotDB closes the state snapshot database without raising an error.
func MustCloseSnapshotDB(db *StateSnapshotDB) {
	if db != nil {
		err := db.Backend.Close()
		if err != nil {
			log.Printf("could not close state snapshot; %s\n", err.Error())
		}
	}
}

// PutCode inserts Account code into database
func (db *StateSnapshotDB) PutCode(code []byte) error {
	// anything to store?
	if len(code) == 0 {
		return nil
	}

	codeHash := crypto.Keccak256Hash(code)
	key := CodeKey(codeHash)
	return db.Backend.Put(key, code)
}

// PutAccount inserts Account into database
func (db *StateSnapshotDB) PutAccount(acc *types.Account) error {
	enc, err := rlp.EncodeToBytes(acc.ToStoredAccount())
	if err != nil {
		return fmt.Errorf("failed encoding account %s to RLP; %s", acc.Hash.String(), err.Error())
	}

	return db.Backend.Put(acc.Hash.Bytes(), enc)
}

// GetAccount retrieves Account from database
func (db *StateSnapshotDB) GetAccount(key common.Hash) (*types.StoredAccount, error) {
	data, err := db.Backend.Get(key.Bytes())
	if err != nil {
		return nil, err
	}
	var acc = types.StoredAccount{}
	err = rlp.DecodeBytes(data, &acc)
	return &acc, err
}

// CodeKey retrieves storing DB key for supplied codeHash
func CodeKey(codeHash common.Hash) []byte {
	prefix := []byte(CodePrefix)
	return append(prefix, codeHash.Bytes()...)
}

// PutBlockNumber inserts block number into database
func (db *StateSnapshotDB) PutBlockNumber(i uint64) error {
	enc, err := rlp.EncodeToBytes(i)
	if err != nil {
		return fmt.Errorf("failed encoding blockID %d to RLP; %s", i, err.Error())
	}

	return db.Backend.Put(BlockNumberKey, enc)
}

// GetBlockNumber retrieves block number from database
func (db *StateSnapshotDB) GetBlockNumber() (uint64, error) {
	data, err := db.Backend.Get(BlockNumberKey)
	if err != nil {
		return 0, fmt.Errorf("block number not found in database; %s", err.Error())
	}

	var blockNumber uint64
	err = rlp.DecodeBytes(data, &blockNumber)
	if err != nil {
		return 0, fmt.Errorf("failed decoding block number from RLP; %s", err.Error())
	}

	return blockNumber, err
}

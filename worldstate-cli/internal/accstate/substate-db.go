package accstate

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"io"
	"log"
)

const (
	CodePrefix = "1c" // CodePrefix + codeHash (256-bit) -> code
)

type StateDB struct {
	Backend BackendDatabase
}

type BackendDatabase interface {
	ethdb.KeyValueReader
	ethdb.KeyValueWriter
	ethdb.Batcher
	ethdb.Iteratee
	ethdb.Stater
	ethdb.Compacter
	io.Closer
}

// OpenOutputDB opens StateDB database
func OpenOutputDB(subStateDataDir string) (*StateDB, error) {
	backend, err := rawdb.NewLevelDBDatabase(subStateDataDir, 1024, 100, "substatedir", false)
	if err != nil {
		return nil, err
	}

	return &StateDB{Backend: backend}, nil
}

// PutCode inserts Account code into database
func (db *StateDB) PutCode(code []byte) error {
	if len(code) == 0 {
		return nil
	}
	codeHash := crypto.Keccak256Hash(code)
	key := CodeKey(codeHash)
	return db.Backend.Put(key, code)
}

// PutAccount inserts Account into database
func (db *StateDB) PutAccount(acc Account) error {
	accountBytes, err := rlp.EncodeToBytes(acc.StoredAccount())
	if err != nil {
		log.Fatal("Encoding acc to rlp", "addrHash", acc.Hash, "error", err)
	}

	return db.Backend.Put(acc.Hash.Bytes(), accountBytes)
}

// CodeKey retrieves storing DB key for supplied codeHash
func CodeKey(codeHash common.Hash) []byte {
	prefix := []byte(CodePrefix)
	return append(prefix, codeHash.Bytes()...)
}

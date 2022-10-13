// Package snapshot implements database interfaces for the world state manager.
package snapshot

import (
	"bytes"
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
	CodePrefix = 0xc0

	// AccountPrefix is used to store accounts.
	AccountPrefix = 0x0a
)

var (
	// ZeroHash represents an empty hash.
	ZeroHash = common.Hash{}
)

// StateDB represents the state snapshot database handle.
type StateDB struct {
	hashing crypto.KeccakState
	Backend BackendDatabase
}

// BackendDatabase represents the underlying KV store used for the StateDB
type BackendDatabase interface {
	ethdb.KeyValueReader
	ethdb.KeyValueWriter
	ethdb.Batcher
	ethdb.Iteratee
	ethdb.Stater
	ethdb.Compacter
	io.Closer
}

// OpenStateDB opens state snapshot database at the given path.
func OpenStateDB(path string) (*StateDB, error) {
	// use in-memory database?
	if path == "" {
		return &StateDB{Backend: rawdb.NewMemoryDatabase(), hashing: crypto.NewKeccakState()}, nil
	}

	// open file-system DB
	backend, err := rawdb.NewLevelDBDatabase(path, 1024, 100, "aida", false)
	if err != nil {
		return nil, err
	}

	return &StateDB{Backend: backend, hashing: crypto.NewKeccakState()}, nil
}

// MustCloseStateDB closes the state snapshot database without raising an error.
func MustCloseStateDB(db *StateDB) {
	if db != nil {
		err := db.Backend.Close()
		if err != nil {
			log.Printf("could not close state snapshot; %s\n", err.Error())
		}
	}
}

// PutCode inserts Account code into database
func (db *StateDB) PutCode(code []byte) ([]byte, error) {
	// anything to store?
	if code == nil {
		return types.EmptyCode, nil
	}

	codeHash := crypto.Keccak256Hash(code)
	return codeHash.Bytes(), db.Backend.Put(CodeKey(codeHash), code)
}

// Code loads account code from the database, if available.
func (db *StateDB) Code(h common.Hash) ([]byte, error) {
	return db.Backend.Get(CodeKey(h))
}

// PutAccount inserts Account into database
func (db *StateDB) PutAccount(acc *types.Account) error {
	var err error

	// store the code, if any
	acc.CodeHash, err = db.PutCode(acc.Code)
	if err != nil {
		return err
	}

	// encode the account itself
	enc, err := rlp.EncodeToBytes(acc)
	if err != nil {
		return fmt.Errorf("failed encoding account %s to RLP; %s", acc.Hash.String(), err.Error())
	}

	return db.Backend.Put(AccountKey(acc.Hash), enc)
}

// Account tries to read details of the given account address.
func (db *StateDB) Account(addr common.Address) (*types.Account, error) {
	key := AccountKey(crypto.HashData(db.hashing, addr.Bytes()))
	data, err := db.Backend.Get(key)
	if err != nil {
		return nil, err
	}

	return db.decodeAccount(key, data)
}

// decodeAccount decodes an account from state snapshot DB for the given account key and data.
func (db *StateDB) decodeAccount(key []byte, data []byte) (*types.Account, error) {
	acc := types.Account{}
	err := rlp.DecodeBytes(data, &acc)
	if err != nil {
		return nil, err
	}

	// update the account hash
	acc.Hash.SetBytes(key[1:])

	// any code to be loaded?
	if !bytes.Equal(acc.CodeHash, ZeroHash.Bytes()) && !bytes.Equal(acc.CodeHash, types.EmptyCode) {
		codeHash := common.Hash{}
		codeHash.SetBytes(acc.CodeHash)

		acc.Code, err = db.Code(codeHash)
		if err != nil {
			return nil, fmt.Errorf("contract code not found; %s", err.Error())
		}
	}

	return &acc, nil
}

// CodeKey retrieves storing DB key of a code for supplied codeHash
func CodeKey(codeHash common.Hash) []byte {
	key := make([]byte, common.HashLength+1)
	key[0] = CodePrefix
	copy(key[1:], codeHash.Bytes())

	return key
}

// AccountKey retrieves storing DB key of an account for supplied hash
func AccountKey(hash common.Hash) []byte {
	key := make([]byte, common.HashLength+1)
	key[0] = AccountPrefix
	copy(key[1:], hash.Bytes())

	return key
}

// Copy creates a copy of the state snapshot database to the given output handle.
// The copy does not erase previous data from the target database.
// If you want a clean copy, make sure you use an empty DB.
func (db *StateDB) Copy(to *StateDB, onAccount func(*types.Account)) error {
	// make a buffer for reader/writer account exchange
	wb := make(chan types.Account, 100)
	defer func() {
		close(wb)
	}()

	// store data to the target database
	wFail := NewQueueWriter(to, wb)

	// we will use iterator to get all the source accounts
	it := db.NewAccountIterator()
	defer it.Release()

	// iterate source database
	for it.Next() {
		acc := it.Value()
		if it.Error() != nil {
			break
		}

		select {
		case err := <-wFail:
			if err != nil {
				return err
			}
		case wb <- *acc:
			if onAccount != nil {
				onAccount(acc)
			}
		}
	}

	// release resources
	return it.Error()
}

// NewQueueWriter creates a writer thread, which inserts Accounts from an input queue into the given database.
func NewQueueWriter(db *StateDB, in chan types.Account) chan error {
	e := make(chan error, 1)

	go func(fail chan error) {
		defer close(fail)
		for {
			// get all the found accounts from the input channel
			account, open := <-in
			if !open {
				break
			}

			// insert account data
			err := db.PutAccount(&account)
			if err != nil {
				fail <- fmt.Errorf("can not write account %s; %s\n", account.Hash.String(), err.Error())
				return
			}
		}
	}(e)

	return e
}

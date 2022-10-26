// Package snapshot implements database interfaces for the world state manager.
package snapshot

import (
	"bytes"
	"fmt"
	"github.com/Fantom-foundation/Aida/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"io"
	"log"
)

var (
	// ZeroHash represents an empty hash.
	ZeroHash = common.Hash{}

	// CodePrefix represents a prefix added to the code hash to separate code and state data in the KV database.
	// CodePrefix + codeHash (256-bit) -> code
	CodePrefix = []byte{0xc0, 0xcc, 0xcc, 0xcc}

	// AccountPrefix is used to store accounts.
	AccountPrefix = []byte{0x0a, 0xaa, 0xaa, 0xaa}

	// BlockNumberKey is key under which block number is stored in database
	BlockNumberKey = []byte{0xbb, 0xbb, 0xbb, 0xbb}

	// HashToAddressPrefix is a prefix used to store and retrieve hash to account address
	HashToAddressPrefix = []byte{0x68, 0x32, 0x61, 0x2d}
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
	return db.AccountByHash(crypto.HashData(db.hashing, addr.Bytes()))
}

// AccountByHash tries to read details of the given account by the account hash.
func (db *StateDB) AccountByHash(hash common.Hash) (*types.Account, error) {
	key := AccountKey(hash)
	data, err := db.Backend.Get(key)
	if err != nil {
		return nil, fmt.Errorf("key %s not found; %s", common.Bytes2Hex(key), err.Error())
	}

	return db.decodeAccount(key, data)
}

// decodeAccount decodes an account from state snapshot DB for the given account key and data.
func (db *StateDB) decodeAccount(key []byte, data []byte) (*types.Account, error) {
	acc := types.Account{}
	err := rlp.DecodeBytes(data, &acc)
	if err != nil {
		return nil, fmt.Errorf("can not decode account; %s", err.Error())
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
	key := make([]byte, common.HashLength+4)
	copy(key[:4], CodePrefix)
	copy(key[4:], codeHash.Bytes())

	return key
}

// AccountKey retrieves storing DB key of an account for supplied hash
func AccountKey(hash common.Hash) []byte {
	key := make([]byte, common.HashLength+4)
	copy(key[:4], AccountPrefix)
	copy(key[4:], hash.Bytes())

	return key
}

// PutBlockNumber inserts block number into database
func (db *StateDB) PutBlockNumber(i uint64) error {
	enc, err := rlp.EncodeToBytes(i)
	if err != nil {
		return fmt.Errorf("failed encoding blockID %d to RLP; %s", i, err.Error())
	}

	return db.Backend.Put(BlockNumberKey, enc)
}

// GetBlockNumber retrieves block number from database
func (db *StateDB) GetBlockNumber() (uint64, error) {
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

// PutHashToAccountAddress puts hash to account address mapping records into the database.
func (db *StateDB) PutHashToAccountAddress(hash common.Hash, adr common.Address) error {
	return db.Backend.Put(Hash2AddressKey(hash), adr.Bytes())
}

// HashToAccountAddress tries to find account address mapped to the specified hash in the state DB.
func (db *StateDB) HashToAccountAddress(hash common.Hash) (common.Address, error) {
	var adr common.Address

	// read mapping data from the database
	data, err := db.Backend.Get(Hash2AddressKey(hash))
	if err != nil {
		return adr, err
	}

	adr.SetBytes(data)
	return adr, nil
}

// AccountAddressToHash returns account address hash for the given address.
func (db *StateDB) AccountAddressToHash(adr common.Address) common.Hash {
	return crypto.HashData(db.hashing, adr.Bytes())
}

// Hash2AddressKey generates storage key for hash -> account address mapping.
// We assume 4 bytes prefix + 32 bytes hash.
func Hash2AddressKey(h common.Hash) []byte {
	k := make([]byte, 36)
	copy(k[:4], HashToAddressPrefix)
	copy(k[4:], h.Bytes())
	return k
}

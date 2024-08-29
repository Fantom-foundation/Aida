// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package state

// A legacy code for substate-cli
import (
	"errors"
	"fmt"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-ethereum/triedb"
)

// offTheChainDB is state.cachingDB clone without disk caches
type offTheChainDB struct {
	trie *triedb.Database
	disk ethdb.Database
	// State witness if cross validation is needed
	witness *stateless.Witness
}

// OpenTrie opens the main account trie at a specific root hash.
func (db *offTheChainDB) OpenTrie(root common.Hash) (state.Trie, error) {
	tr, err := trie.NewStateTrie(trie.StateTrieID(root), db.trie)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// OpenStorageTrie opens the storage trie of an account.
func (db *offTheChainDB) OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, _ state.Trie) (state.Trie, error) {
	tr, err := trie.NewStateTrie(trie.StorageTrieID(stateRoot, crypto.Keccak256Hash(address.Bytes()), root), db.trie)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// CopyTrie returns an independent copy of the given trie.
func (db *offTheChainDB) CopyTrie(t state.Trie) state.Trie {
	switch t := t.(type) {
	case *trie.StateTrie:
		return t.Copy()
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}

// ContractCode retrieves a particular contract's code.
func (db *offTheChainDB) ContractCode(address common.Address, codeHash common.Hash) ([]byte, error) {
	code := rawdb.ReadCode(db.disk, codeHash)
	if len(code) > 0 {
		return code, nil
	}
	return nil, errors.New("not found")
}

// ContractCodeWithPrefix retrieves a particular contract's code. If the
// code can't be found in the cache, then check the existence with **new**
// db scheme.
func (db *offTheChainDB) ContractCodeWithPrefix(address common.Address, codeHash common.Hash) ([]byte, error) {
	code := rawdb.ReadCodeWithPrefix(db.disk, codeHash)
	if len(code) > 0 {
		return code, nil
	}
	return nil, errors.New("not found")
}

// ContractCodeSize retrieves a particular contracts code's size.
func (db *offTheChainDB) ContractCodeSize(address common.Address, codeHash common.Hash) (int, error) {
	code, err := db.ContractCode(address, codeHash)
	return len(code), err
}

// DiskDB returns the underlying key-value disk database.
func (db *offTheChainDB) DiskDB() ethdb.KeyValueStore {
	return db.disk
}

// TrieDB retrieves any intermediate trie-node caching layer.
func (db *offTheChainDB) TrieDB() *triedb.Database {
	return db.trie
}

func (db *offTheChainDB) PointCache() *utils.PointCache {
	// this should not be relevant for revisions up to Cancun
	panic("PointCache not implemented")
}

// Witness retrieves the current state witness being collected.
func (db *offTheChainDB) Witness() *stateless.Witness {
	// this should not be relevant for revisions up to Cancun
	return nil
}

// NewOffTheChainStateDB returns an empty in-memory *state.StateDB without disk caches
func NewOffTheChainStateDB() *state.StateDB {
	// backend in-memory key-value database
	kvdb := rawdb.NewMemoryDatabase()

	// zeroed trie.Config to disable Cache, Journal, Preimages, ...
	zerodConfig := &triedb.Config{}
	tdb := triedb.NewDatabase(kvdb, zerodConfig)

	sdb := &offTheChainDB{
		trie: tdb,
		disk: kvdb,
	}

	statedb, err := state.New(types.EmptyRootHash, sdb, nil)
	if err != nil {
		panic(fmt.Errorf("error calling state.New() in NewOffTheChainDB(): %v", err))
	}
	return statedb
}

// MakeOffTheChainStateDB returns an in-memory *state.StateDB initialized with ws
func MakeOffTheChainStateDB(alloc txcontext.WorldState, block uint64, chainConduit *ChainConduit) (StateDB, error) {
	statedb := NewOffTheChainStateDB()
	alloc.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
		code := acc.GetCode()
		statedb.SetCode(addr, code)
		statedb.SetNonce(addr, acc.GetNonce())
		statedb.SetBalance(addr, acc.GetBalance(), 0)
		// DON'T USE SetStorage because it makes REVERT and dirtyStorage unavailble
		acc.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
			statedb.SetState(addr, keyHash, valueHash)
		})
	})

	// Commit and re-open to start with a clean state.
	_, err := statedb.Commit(block, false)
	if err != nil {
		return nil, fmt.Errorf("cannot commit offTheChainDb; %v", err)
	}

	return &gethStateDB{db: statedb, block: block, chainConduit: chainConduit}, nil
}

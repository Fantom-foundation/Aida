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

import (
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Aida/txcontext"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/core/rawdb"
	geth "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
)

const (
	triesInMemory    = 1
	memoryUpperLimit = 256 * 1024 * 1024
	imgUpperLimit    = 4 * 1024 * 1024
)

func MakeGethStateDB(directory, variant string, rootHash common.Hash, isArchiveMode bool, chainConduit *ChainConduit) (StateDB, error) {
	if variant != "" {
		return nil, fmt.Errorf("unknown variant: %v", variant)
	}
	const cacheSize = 512
	const fileHandle = 128
	ldb, err := rawdb.NewLevelDBDatabase(directory, cacheSize, fileHandle, "", false)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new Level DB. %v", err)
	}
	evmState := geth.NewDatabase(ldb)
	db, err := geth.New(rootHash, evmState, nil)
	if err != nil {
		return nil, err
	}

	return &gethStateDB{
		db:            db,
		evmState:      evmState,
		stateRoot:     rootHash,
		triegc:        prque.New(nil),
		isArchiveMode: isArchiveMode,
		chainConduit:  chainConduit,
	}, nil
}

// openStateDB creates a new statedb from an existing geth database
func (s *gethStateDB) openStateDB() error {
	var err error
	s.db, err = geth.NewWithSnapLayers(s.stateRoot, s.evmState, nil, 0)
	return err
}

type gethStateDB struct {
	db            vm.StateDB    // statedb
	evmState      geth.Database // key-value database
	stateRoot     common.Hash   // lastest root hash
	triegc        *prque.Prque
	isArchiveMode bool
	chainConduit  *ChainConduit // chain configuration
	block         *big.Int
}

func (s *gethStateDB) CreateAccount(addr common.Address) {
	s.db.CreateAccount(addr)
}

func (s *gethStateDB) Exist(addr common.Address) bool {
	return s.db.Exist(addr)
}

func (s *gethStateDB) Empty(addr common.Address) bool {
	return s.db.Empty(addr)
}

func (s *gethStateDB) Suicide(addr common.Address) bool {
	return s.db.Suicide(addr)
}

func (s *gethStateDB) HasSuicided(addr common.Address) bool {
	return s.db.HasSuicided(addr)
}

func (s *gethStateDB) GetBalance(addr common.Address) *big.Int {
	return s.db.GetBalance(addr)
}

func (s *gethStateDB) AddBalance(addr common.Address, value *big.Int) {
	s.db.AddBalance(addr, value)
}

func (s *gethStateDB) SubBalance(addr common.Address, value *big.Int) {
	s.db.SubBalance(addr, value)
}

func (s *gethStateDB) GetNonce(addr common.Address) uint64 {
	return s.db.GetNonce(addr)
}

func (s *gethStateDB) SetNonce(addr common.Address, value uint64) {
	s.db.SetNonce(addr, value)
}

func (s *gethStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	return s.db.GetCommittedState(addr, key)
}

func (s *gethStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	return s.db.GetState(addr, key)
}

func (s *gethStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.db.SetState(addr, key, value)
}

func (s *gethStateDB) GetCode(addr common.Address) []byte {
	return s.db.GetCode(addr)
}

func (s *gethStateDB) GetCodeHash(addr common.Address) common.Hash {
	return s.db.GetCodeHash(addr)
}

func (s *gethStateDB) GetCodeSize(addr common.Address) int {
	return s.db.GetCodeSize(addr)
}

func (s *gethStateDB) SetCode(addr common.Address, code []byte) {
	s.db.SetCode(addr, code)
}

func (s *gethStateDB) Snapshot() int {
	return s.db.Snapshot()
}

func (s *gethStateDB) RevertToSnapshot(id int) {
	s.db.RevertToSnapshot(id)
}

func (s *gethStateDB) Error() error {
	// TODO return geth's dberror
	return nil
}

func (s *gethStateDB) BeginTransaction(number uint32) error {
	// ignored
	return nil
}

func (s *gethStateDB) EndTransaction() error {
	if s.chainConduit == nil || s.chainConduit.IsFinalise(s.block) {
		// Opera or Ethereum after Byzantium
		s.Finalise(true)
	} else {
		// Ethereum before Byzantium
		s.IntermediateRoot(s.chainConduit.DeleteEmptyObjects(s.block))
	}
	return nil
}

func (s *gethStateDB) BeginBlock(number uint64) error {
	if err := s.openStateDB(); err != nil {
		return fmt.Errorf("cannot open geth state-db; %w", err)
	}
	s.block = new(big.Int).SetUint64(number)
	return nil
}

func (s *gethStateDB) EndBlock() error {
	var err error
	//commit at the end of a block
	s.stateRoot, err = s.Commit(true)
	if err != nil {
		panic("StateDB commit failed")
	}
	// if archival node, flush trie to disk after each block
	if s.evmState != nil {
		if err = s.trieCommit(); err != nil {
			return fmt.Errorf("cannot commit trie; %w", err)
		}
		s.trieCap()
	}
	return nil
}

func (s *gethStateDB) BeginSyncPeriod(number uint64) {
	// ignored
}

func (s *gethStateDB) EndSyncPeriod() {
	// if not archival node, flush trie to disk after each sync-period
	if s.evmState != nil && !s.isArchiveMode {
		s.trieCleanCommit()
		s.trieCap()
	}
}

func (s *gethStateDB) GetHash() (common.Hash, error) {
	return s.IntermediateRoot(true), nil
}

func (s *gethStateDB) Finalise(deleteEmptyObjects bool) {
	if db, ok := s.db.(*geth.StateDB); ok {
		db.Finalise(deleteEmptyObjects)
	}
}

func (s *gethStateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	if db, ok := s.db.(*geth.StateDB); ok {
		return db.IntermediateRoot(deleteEmptyObjects)
	}
	return common.Hash{}
}

func (s *gethStateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	if db, ok := s.db.(*geth.StateDB); ok {
		return db.Commit(deleteEmptyObjects)
	}
	return common.Hash{}, nil
}

func (s *gethStateDB) Prepare(thash common.Hash, ti int) {
	if db, ok := s.db.(*geth.StateDB); ok {
		db.Prepare(thash, ti)
	}
}

func (s *gethStateDB) PrepareSubstate(substate txcontext.WorldState, block uint64) {
	// ignored
}

func (s *gethStateDB) GetSubstatePostAlloc() txcontext.WorldState {
	if db, ok := s.db.(*geth.StateDB); ok {
		return substatecontext.NewWorldState(db.GetSubstatePostAlloc())
	}

	return nil
}

func (s *gethStateDB) Close() error {
	// Skip closing if implementation is not Geth based.
	state, ok := s.db.(*geth.StateDB)
	if !ok {
		return nil
	}
	// Commit data to trie.
	hash, err := state.Commit(true)
	if err != nil {
		return err
	}

	// Close underlying trie caching intermediate results.
	db := state.Database().TrieDB()
	if err := db.Commit(hash, true, nil); err != nil {
		return err
	}

	// Close underlying LevelDB instance.
	return db.DiskDB().Close()
}

func (s *gethStateDB) AddRefund(gas uint64) {
	s.db.AddRefund(gas)
}

func (s *gethStateDB) SubRefund(gas uint64) {
	s.db.SubRefund(gas)
}
func (s *gethStateDB) GetRefund() uint64 {
	return s.db.GetRefund()
}
func (s *gethStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.db.PrepareAccessList(sender, dest, precompiles, txAccesses)
}

func (s *gethStateDB) AddressInAccessList(addr common.Address) bool {
	return s.db.AddressInAccessList(addr)
}
func (s *gethStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return s.db.SlotInAccessList(addr, slot)
}
func (s *gethStateDB) AddAddressToAccessList(addr common.Address) {
	s.db.AddAddressToAccessList(addr)
}
func (s *gethStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.db.AddSlotToAccessList(addr, slot)
}

func (s *gethStateDB) AddLog(log *types.Log) {
	s.db.AddLog(log)
}
func (s *gethStateDB) AddPreimage(hash common.Hash, preimage []byte) {
	panic("Add Preimage")
	s.db.AddPreimage(hash, preimage)
}
func (s *gethStateDB) ForEachStorage(addr common.Address, cb func(common.Hash, common.Hash) bool) error {
	return s.db.ForEachStorage(addr, cb)
}
func (s *gethStateDB) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	if db, ok := s.db.(*geth.StateDB); ok {
		return db.GetLogs(hash, blockHash)
	}
	return []*types.Log{}
}

func (s *gethStateDB) StartBulkLoad(block uint64) (BulkLoad, error) {
	if err := s.BeginBlock(block); err != nil {
		return nil, err
	}
	if err := s.BeginTransaction(0); err != nil {
		return nil, err
	}
	return &gethBulkLoad{db: s}, nil
}

func (s *gethStateDB) GetArchiveState(block uint64) (NonCommittableStateDB, error) {
	return nil, fmt.Errorf("archive states are not (yet) supported by this DB implementation")
}

func (s *gethStateDB) GetArchiveBlockHeight() (uint64, bool, error) {
	return 0, false, fmt.Errorf("archive states are not (yet) supported by this DB implementation")
}

func (s *gethStateDB) GetMemoryUsage() *MemoryUsage {
	// not supported yet
	return &MemoryUsage{uint64(0), nil}
}

type gethBulkLoad struct {
	db *gethStateDB
}

func (l *gethBulkLoad) CreateAccount(addr common.Address) {
	l.db.CreateAccount(addr)
}

func (l *gethBulkLoad) SetBalance(addr common.Address, value *big.Int) {
	old := l.db.GetBalance(addr)
	value = value.Sub(value, old)
	l.db.AddBalance(addr, value)
}

func (l *gethBulkLoad) SetNonce(addr common.Address, nonce uint64) {
	l.db.SetNonce(addr, nonce)
}

func (l *gethBulkLoad) SetState(addr common.Address, key common.Hash, value common.Hash) {
	l.db.SetState(addr, key, value)
}

func (l *gethBulkLoad) SetCode(addr common.Address, code []byte) {
	l.db.SetCode(addr, code)
}

func (l *gethBulkLoad) Close() error {
	l.db.EndTransaction()
	l.db.EndBlock()
	_, err := l.db.Commit(false)
	return err
}

// trieCommit commits changes to disk if archive node; otherwise, performs garbage collection.
func (s *gethStateDB) trieCommit() error {
	triedb := s.evmState.TrieDB()
	// If we're applying genesis or running an archive node, always flush
	if s.isArchiveMode {
		if err := triedb.Commit(s.stateRoot, false, nil); err != nil {
			return fmt.Errorf("Failed to flush trie DB into main DB. %v", err)
		}
	} else {
		// Full but not archive node, do proper garbage collection
		triedb.Reference(s.stateRoot, common.Hash{}) // metadata reference to keep trie alive
		s.triegc.Push(s.stateRoot, -int64(s.block.Uint64()))

		if current := s.block.Uint64(); current > triesInMemory {
			// If we exceeded our memory allowance, flush matured singleton nodes to disk
			s.trieCap()

			// Find the next state trie we need to commit
			chosen := current - triesInMemory

			// Garbage collect all below the chosen block
			for !s.triegc.Empty() {
				root, number := s.triegc.Pop()
				if uint64(-number) > chosen {
					s.triegc.Push(root, number)
					break
				}
				triedb.Dereference(root.(common.Hash))
			}
		}
	}
	return nil
}

// trieCleanCommit cleans old state trie and commit changes.
func (s *gethStateDB) trieCleanCommit() error {
	// Don't need to reference the current state root
	// due to it already be referenced on `Commit()` function
	triedb := s.evmState.TrieDB()
	if current := s.block.Uint64(); current > triesInMemory {
		// Find the next state trie we need to commit
		chosen := current - triesInMemory
		// Garbage collect all below the chosen block
		for !s.triegc.Empty() {
			root, number := s.triegc.Pop()
			if uint64(-number) > chosen {
				s.triegc.Push(root, number)
				break
			}
			triedb.Dereference(root.(common.Hash))
		}
	}
	// commit the state trie after clean up
	err := triedb.Commit(s.stateRoot, false, nil)
	return err
}

// trieCap flushes matured singleton nodes to disk.
func (s *gethStateDB) trieCap() {
	triedb := s.evmState.TrieDB()
	nodes, imgs := triedb.Size()
	if nodes > memoryUpperLimit+ethdb.IdealBatchSize || imgs > imgUpperLimit {
		//If we exceeded our memory allowance, flush matured singleton nodes to disk
		triedb.Cap(memoryUpperLimit)
	}
}

func (s *gethStateDB) GetShadowDB() StateDB {
	return nil
}

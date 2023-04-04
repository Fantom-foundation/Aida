package state

import (
	"fmt"
	"math/big"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
	rawdb "github.com/ethereum/go-ethereum/core/rawdb"
	geth "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	vm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"

	estate "github.com/ledgerwatch/erigon/core/state"
	erigonethdb "github.com/ledgerwatch/erigon/ethdb"

	"github.com/ledgerwatch/erigon-lib/kv"
)

const (
	triesInMemory    = 16
	memoryUpperLimit = 256 * 1024 * 1024
	imgUpperLimit    = 4 * 1024 * 1024
)

func MakeGethStateDB(directory, variant string, rootHash common.Hash, isArchiveMode bool) (StateDB, error) {
	if variant != "" {
		return nil, fmt.Errorf("unkown variant: %v", variant)
	}
	const cacheSize = 512
	const fileHandle = 128
	ldb, err := rawdb.NewLevelDBDatabase(directory, cacheSize, fileHandle, "", false)
	if err != nil {
		return nil, fmt.Errorf("Failed to create a new Level DB. %v", err)
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
	}, nil
}

// BeginBlockApply creates a new statedb from an existing geth database
func (s *gethStateDB) BeginBlockApply() error {
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
	block         uint64
}

func (s *gethStateDB) SetTxBlock(uint64) {}

func (s *gethStateDB) DB() erigonethdb.Database { return nil }

func (s *gethStateDB) CommitBlock(stateWriter estate.StateWriter) error { return nil }

func (s *gethStateDB) CommitBlockWithStateWriter() error { return nil }

func (s *gethStateDB) NewBatch(kv.RwTx, chan struct{}) erigonethdb.DbWithPendingMutations { return nil }

func (s *gethStateDB) BeginBlockApplyBatch(batch erigonethdb.DbWithPendingMutations, noHistory bool, rwTx kv.RwTx) error {
	return nil
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

func (s *gethStateDB) BeginTransaction(number uint32) {
	// ignored
}

func (s *gethStateDB) EndTransaction() {
	// ignored
}

func (s *gethStateDB) BeginBlock(number uint64) {
	s.block = number
}

func (s *gethStateDB) EndBlock() {
	var err error
	//commit at the end of a block
	s.stateRoot, err = s.Commit(true)
	if err != nil {
		panic(fmt.Errorf("StateDB commit failed\n"))
	}
	// if archival node, flush trie to disk after each block
	if s.evmState != nil {
		s.trieCommit()
		s.trieCap()
	}
}

func (s *gethStateDB) BeginEpoch(number uint64) {
	// ignored
}

func (s *gethStateDB) EndEpoch() {
	// if not archival node, flush trie to disk after each epoch
	if s.evmState != nil && !s.isArchiveMode {
		s.trieCleanCommit()
		s.trieCap()
	}
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

func (s *gethStateDB) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	// ignored
}

func (s *gethStateDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	if db, ok := s.db.(*geth.StateDB); ok {
		return db.GetSubstatePostAlloc()
	}
	return substate.SubstateAlloc{}
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

func (s *gethStateDB) StartBulkLoad() BulkLoad {
	return &gethBulkLoad{db: s}
}

func (s *gethStateDB) GetArchiveState(block uint64) (StateDB, error) {
	return nil, fmt.Errorf("archive states are not (yet) supported by this DB implementation")
}

func (s *gethStateDB) GetMemoryUsage() *MemoryUsage {
	// not supported yet
	return nil
}

type gethBulkLoad struct {
	db      *gethStateDB
	num_ops int64
}

func (l *gethBulkLoad) CreateAccount(addr common.Address) {
	l.db.CreateAccount(addr)
	l.digest()
}

func (l *gethBulkLoad) SetBalance(addr common.Address, value *big.Int) {
	old := l.db.GetBalance(addr)
	value = value.Sub(value, old)
	l.db.AddBalance(addr, value)
	l.digest()
}

func (l *gethBulkLoad) SetNonce(addr common.Address, nonce uint64) {
	l.db.SetNonce(addr, nonce)
	l.digest()
}

func (l *gethBulkLoad) SetState(addr common.Address, key common.Hash, value common.Hash) {
	l.db.SetState(addr, key, value)
	l.digest()
}

func (l *gethBulkLoad) SetCode(addr common.Address, code []byte) {
	l.db.SetCode(addr, code)
	l.digest()
}

func (l *gethBulkLoad) Close() error {
	l.db.EndBlock()
	l.db.EndEpoch()
	_, err := l.db.Commit(false)
	return err
}

func (l *gethBulkLoad) digest() {
	// Call EndBlock every 1M insert operation.
	l.num_ops++
	if l.num_ops%(1000*1000) != 0 {
		return
	}
	l.db.EndBlock()
}

// tireCommit commits changes to disk if archive node; otherwise, performs garbage collection.
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
		s.triegc.Push(s.stateRoot, -int64(s.block))

		if current := uint64(s.block); current > triesInMemory {
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
	if current := uint64(s.block); current > triesInMemory {
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

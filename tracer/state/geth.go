package state

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	rawdb "github.com/ethereum/go-ethereum/core/rawdb"
	geth "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	vm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/substate"
)

func MakeGethStateDB(directory, variant string) (StateDB, error) {
	if variant != "" {
		return nil, fmt.Errorf("unkown variant: %v", variant)
	}
	return OpenGethStateDB(directory, common.Hash{})
}

func OpenGethStateDB(directory string, root_hash common.Hash) (StateDB, error) {
	const cache_size = 512
	const file_handle = 128
	ldb, err := rawdb.NewLevelDBDatabase(directory, cache_size, file_handle, "", false)
	if err != nil {
		return nil, fmt.Errorf("Failed to create a new Level DB. %v", err)
	}
	ethdb := geth.NewDatabase(ldb)
	db, err := geth.New(root_hash, ethdb, nil)
	if err != nil {
		return nil, err
	}
	return &gethStateDB{db: db, ethdb: ethdb}, nil
}

// BeginBlockApply creates a new statedb from an existing geth database
func (s *gethStateDB) BeginBlockApply(root_hash common.Hash) error {
	var err error
	s.db, err = geth.NewWithSnapLayers(root_hash, s.ethdb, nil, 0)
	return err
}

type gethStateDB struct {
	db    vm.StateDB
	ethdb geth.Database
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

func (s *gethStateDB) BeginTransaction(number uint32) {
	// ignored
}

func (s *gethStateDB) EndTransaction() {
	// ignored
}

func (s *gethStateDB) BeginBlock(number uint64) {
	// ignored
}

func (s *gethStateDB) EndBlock() {
	//commit at the end of a block
	if _, err := s.Commit(true); err != nil {
		panic(fmt.Errorf("StateDB commit failed\n"))
	}
}

func (s *gethStateDB) BeginEpoch(number uint64) {
	// ignored
}

func (s *gethStateDB) EndEpoch() {
	// ignored
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

func (s *gethStateDB) PrepareSubstate(substate *substate.SubstateAlloc) {
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

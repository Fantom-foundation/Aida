package state

import (
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/substate"
)

// MakeLoggingStateDB wrapps the given StateDB instance into a logging wrapper causing
// every StateDB operation (except BulkLoading) to be logged for debugging.
func MakeLoggingStateDB(db StateDB) StateDB {
	return &loggingStateDB{db}
}

type loggingStateDB struct {
	db StateDB
}

func (s *loggingStateDB) BeginBlockApply() error {
	log.Printf("BeginBlockApply\n")
	return s.db.BeginBlockApply()
}

func (s *loggingStateDB) CreateAccount(addr common.Address) {
	log.Printf("CreateAccount, %v\n", addr)
	s.db.CreateAccount(addr)
}

func (s *loggingStateDB) Exist(addr common.Address) bool {
	res := s.db.Exist(addr)
	log.Printf("Exist, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) Empty(addr common.Address) bool {
	res := s.db.Empty(addr)
	log.Printf("Empty, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) Suicide(addr common.Address) bool {
	res := s.db.Suicide(addr)
	log.Printf("Suicide, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) HasSuicided(addr common.Address) bool {
	res := s.db.HasSuicided(addr)
	log.Printf("HasSuicided, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) GetBalance(addr common.Address) *big.Int {
	res := s.db.GetBalance(addr)
	log.Printf("GetBalance, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) AddBalance(addr common.Address, value *big.Int) {
	s.db.AddBalance(addr, value)
	log.Printf("AddBalance, %v, %v, %v\n", addr, value, s.db.GetBalance(addr))
}

func (s *loggingStateDB) SubBalance(addr common.Address, value *big.Int) {
	s.db.SubBalance(addr, value)
	log.Printf("SubBalance, %v, %v, %v\n", addr, value, s.db.GetBalance(addr))
}

func (s *loggingStateDB) GetNonce(addr common.Address) uint64 {
	res := s.db.GetNonce(addr)
	log.Printf("GetNonce, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) SetNonce(addr common.Address, value uint64) {
	s.db.SetNonce(addr, value)
	log.Printf("SetNonce, %v, %v\n", addr, value)
}

func (s *loggingStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	res := s.db.GetCommittedState(addr, key)
	log.Printf("GetCommittedState, %v, %v, %v\n", addr, key, res)
	return res
}

func (s *loggingStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	res := s.db.GetState(addr, key)
	log.Printf("GetState, %v, %v, %v\n", addr, key, res)
	return res
}

func (s *loggingStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.db.SetState(addr, key, value)
	log.Printf("SetState, %v, %v, %v\n", addr, key, value)
}

func (s *loggingStateDB) GetCode(addr common.Address) []byte {
	res := s.db.GetCode(addr)
	log.Printf("GetCode, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) GetCodeSize(addr common.Address) int {
	res := s.db.GetCodeSize(addr)
	log.Printf("GetCodeSize, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) GetCodeHash(addr common.Address) common.Hash {
	res := s.db.GetCodeHash(addr)
	log.Printf("GetCodeHash, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) SetCode(addr common.Address, code []byte) {
	s.db.SetCode(addr, code)
	log.Printf("SetCode, %v, %v\n", addr, code)
}

func (s *loggingStateDB) Snapshot() int {
	res := s.db.Snapshot()
	log.Printf("Snapshot, %v\n", res)
	return res
}

func (s *loggingStateDB) RevertToSnapshot(id int) {
	s.db.RevertToSnapshot(id)
	log.Printf("RevertToSnapshot, %v\n", id)
}

func (s *loggingStateDB) BeginTransaction(tx uint32) {
	log.Printf("BeginTransaction, %v\n", tx)
	s.db.BeginTransaction(tx)
}

func (s *loggingStateDB) EndTransaction() {
	log.Printf("EndTransaction\n")
	s.db.EndTransaction()
}

func (s *loggingStateDB) BeginBlock(blk uint64) {
	log.Printf("BeginBlock, %v\n", blk)
	s.db.BeginBlock(blk)
}

func (s *loggingStateDB) EndBlock() {
	log.Printf("EndBlock\n")
	s.db.EndBlock()
}

func (s *loggingStateDB) BeginEpoch(number uint64) {
	log.Printf("BeginEpoch, %v\n", number)
	s.db.BeginEpoch(number)
}

func (s *loggingStateDB) EndEpoch() {
	log.Printf("EndEpoch\n")
	s.db.EndEpoch()
}

func (s *loggingStateDB) Close() error {
	res := s.db.Close()
	log.Printf("EndEpoch, %v\n", res)
	return res
}

func (s *loggingStateDB) AddRefund(amount uint64) {
	s.db.AddRefund(amount)
	log.Printf("AddRefund, %v, %v\n", amount, s.db.GetRefund())
}

func (s *loggingStateDB) SubRefund(amount uint64) {
	s.db.SubRefund(amount)
	log.Printf("SubRefund, %v, %v\n", amount, s.db.GetRefund())
}

func (s *loggingStateDB) GetRefund() uint64 {
	res := s.db.GetRefund()
	log.Printf("GetRefund, %v\n", res)
	return res
}

func (s *loggingStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	log.Printf("PrepareAccessList, %v, %v, %v, %v\n", sender, dest, precompiles, txAccesses)
	s.db.PrepareAccessList(sender, dest, precompiles, txAccesses)
}

func (s *loggingStateDB) AddressInAccessList(addr common.Address) bool {
	res := s.db.AddressInAccessList(addr)
	log.Printf("AddressInAccessList, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	a, b := s.db.SlotInAccessList(addr, slot)
	log.Printf("SlotInAccessList, %v, %v, %v, %v\n", addr, slot, a, b)
	return a, b
}

func (s *loggingStateDB) AddAddressToAccessList(addr common.Address) {
	log.Printf("AddAddressToAccessList, %v\n", addr)
	s.db.AddAddressToAccessList(addr)
}

func (s *loggingStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	log.Printf("AddSlotToAccessList, %v, %v\n", addr, slot)
	s.db.AddSlotToAccessList(addr, slot)
}

func (s *loggingStateDB) AddLog(entry *types.Log) {
	log.Printf("AddLog, %v\n", entry)
	s.db.AddLog(entry)
}

func (s *loggingStateDB) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	res := s.db.GetLogs(hash, blockHash)
	log.Printf("GetLogs, %v, %v, %v\n", hash, blockHash, res)
	return res
}

func (s *loggingStateDB) Finalise(deleteEmptyObjects bool) {
	log.Printf("Finalise, %v\n", deleteEmptyObjects)
	s.db.Finalise(deleteEmptyObjects)
}

func (s *loggingStateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	res := s.db.IntermediateRoot(deleteEmptyObjects)
	log.Printf("IntermediateRoot, %v, %v\n", deleteEmptyObjects, res)
	return res
}

func (s *loggingStateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	hash, err := s.db.Commit(deleteEmptyObjects)
	log.Printf("Commit, %v, %v, %v\n", deleteEmptyObjects, hash, err)
	return hash, err
}

func (s *loggingStateDB) Prepare(thash common.Hash, ti int) {
	s.db.Prepare(thash, ti)
	log.Printf("Prepare, %v, %v\n", thash, ti)
}

func (s *loggingStateDB) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	s.db.PrepareSubstate(substate, block)
	log.Printf("PrepareSubstate, %v\n", substate)
}

func (s *loggingStateDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	res := s.db.GetSubstatePostAlloc()
	log.Printf("GetSubstatePostAlloc, %v\n", res)
	return res
}

func (s *loggingStateDB) AddPreimage(hash common.Hash, data []byte) {
	s.db.AddPreimage(hash, data)
	log.Printf("AddPreimage, %v, %v\n", hash, data)
}

func (s *loggingStateDB) ForEachStorage(addr common.Address, op func(common.Hash, common.Hash) bool) error {
	// no loggin in this case
	return s.db.ForEachStorage(addr, op)
}

func (s *loggingStateDB) StartBulkLoad() BulkLoad {
	// no loggin in this case
	return s.db.StartBulkLoad()
}
func (s *loggingStateDB) GetMemoryUsage() *MemoryUsage {
	// no loggin in this case
	return s.db.GetMemoryUsage()
}

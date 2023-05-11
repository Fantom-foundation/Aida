package state

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/op/go-logging"
)

// MakeLoggingStateDB wrapps the given StateDB instance into a logging wrapper causing
// every StateDB operation (except BulkLoading) to be logged for debugging.
func MakeLoggingStateDB(db StateDB, cfg *utils.Config) StateDB {
	return &loggingStateDB{
		db:  db,
		log: utils.NewLogger(cfg.LogLevel, "Logging State DB"),
	}
}

type loggingStateDB struct {
	db  StateDB
	log *logging.Logger
}

func (s *loggingStateDB) CreateAccount(addr common.Address) {
	s.db.CreateAccount(addr)
	s.log.Infof("CreateAccount, %v\n", addr)
}

func (s *loggingStateDB) Exist(addr common.Address) bool {
	res := s.db.Exist(addr)
	s.log.Infof("Exist, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) Empty(addr common.Address) bool {
	res := s.db.Empty(addr)
	s.log.Infof("Empty, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) Suicide(addr common.Address) bool {
	res := s.db.Suicide(addr)
	s.log.Infof("Suicide, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) HasSuicided(addr common.Address) bool {
	res := s.db.HasSuicided(addr)
	s.log.Infof("HasSuicided, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) GetBalance(addr common.Address) *big.Int {
	res := s.db.GetBalance(addr)
	s.log.Infof("GetBalance, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) AddBalance(addr common.Address, value *big.Int) {
	s.db.AddBalance(addr, value)
	s.log.Infof("AddBalance, %v, %v, %v\n", addr, value, s.db.GetBalance(addr))
}

func (s *loggingStateDB) SubBalance(addr common.Address, value *big.Int) {
	s.db.SubBalance(addr, value)
	s.log.Infof("SubBalance, %v, %v, %v\n", addr, value, s.db.GetBalance(addr))
}

func (s *loggingStateDB) GetNonce(addr common.Address) uint64 {
	res := s.db.GetNonce(addr)
	s.log.Infof("GetNonce, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) SetNonce(addr common.Address, value uint64) {
	s.db.SetNonce(addr, value)
	s.log.Infof("SetNonce, %v, %v\n", addr, value)
}

func (s *loggingStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	res := s.db.GetCommittedState(addr, key)
	s.log.Infof("GetCommittedState, %v, %v, %v\n", addr, key, res)
	return res
}

func (s *loggingStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	res := s.db.GetState(addr, key)
	s.log.Infof("GetState, %v, %v, %v\n", addr, key, res)
	return res
}

func (s *loggingStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.db.SetState(addr, key, value)
	s.log.Infof("SetState, %v, %v, %v\n", addr, key, value)
}

func (s *loggingStateDB) GetCode(addr common.Address) []byte {
	res := s.db.GetCode(addr)
	s.log.Infof("GetCode, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) GetCodeSize(addr common.Address) int {
	res := s.db.GetCodeSize(addr)
	s.log.Infof("GetCodeSize, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) GetCodeHash(addr common.Address) common.Hash {
	res := s.db.GetCodeHash(addr)
	s.log.Infof("GetCodeHash, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) SetCode(addr common.Address, code []byte) {
	s.db.SetCode(addr, code)
	s.log.Infof("SetCode, %v, %v\n", addr, code)
}

func (s *loggingStateDB) Snapshot() int {
	res := s.db.Snapshot()
	s.log.Infof("Snapshot, %v\n", res)
	return res
}

func (s *loggingStateDB) RevertToSnapshot(id int) {
	s.db.RevertToSnapshot(id)
	s.log.Infof("RevertToSnapshot, %v\n", id)
}

func (s *loggingStateDB) Error() error {
	s.log.Errorf("Error\n")
	return s.db.Error()
}

func (s *loggingStateDB) BeginTransaction(tx uint32) {
	s.log.Infof("BeginTransaction, %v\n", tx)
	s.db.BeginTransaction(tx)
}

func (s *loggingStateDB) EndTransaction() {
	s.log.Infof("EndTransaction\n")
	s.db.EndTransaction()
}

func (s *loggingStateDB) BeginBlock(blk uint64) {
	s.log.Infof("BeginBlock, %v\n", blk)
	s.db.BeginBlock(blk)
}

func (s *loggingStateDB) EndBlock() {
	s.log.Infof("EndBlock\n")
	s.db.EndBlock()
}

func (s *loggingStateDB) BeginSyncPeriod(number uint64) {
	s.log.Infof("BeginSyncPeriod, %v\n", number)
	s.db.BeginSyncPeriod(number)
}

func (s *loggingStateDB) EndSyncPeriod() {
	s.log.Infof("EndSyncPeriod\n")
	s.db.EndSyncPeriod()
}

func (s *loggingStateDB) Close() error {
	res := s.db.Close()
	s.log.Infof("EndSyncPeriod, %v\n", res)
	return res
}

func (s *loggingStateDB) AddRefund(amount uint64) {
	s.db.AddRefund(amount)
	s.log.Infof("AddRefund, %v, %v\n", amount, s.db.GetRefund())
}

func (s *loggingStateDB) SubRefund(amount uint64) {
	s.db.SubRefund(amount)
	s.log.Infof("SubRefund, %v, %v\n", amount, s.db.GetRefund())
}

func (s *loggingStateDB) GetRefund() uint64 {
	res := s.db.GetRefund()
	s.log.Infof("GetRefund, %v\n", res)
	return res
}

func (s *loggingStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.log.Infof("PrepareAccessList, %v, %v, %v, %v\n", sender, dest, precompiles, txAccesses)
	s.db.PrepareAccessList(sender, dest, precompiles, txAccesses)
}

func (s *loggingStateDB) AddressInAccessList(addr common.Address) bool {
	res := s.db.AddressInAccessList(addr)
	s.log.Infof("AddressInAccessList, %v, %v\n", addr, res)
	return res
}

func (s *loggingStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	a, b := s.db.SlotInAccessList(addr, slot)
	s.log.Infof("SlotInAccessList, %v, %v, %v, %v\n", addr, slot, a, b)
	return a, b
}

func (s *loggingStateDB) AddAddressToAccessList(addr common.Address) {
	s.log.Infof("AddAddressToAccessList, %v\n", addr)
	s.db.AddAddressToAccessList(addr)
}

func (s *loggingStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.log.Infof("AddSlotToAccessList, %v, %v\n", addr, slot)
	s.db.AddSlotToAccessList(addr, slot)
}

func (s *loggingStateDB) AddLog(entry *types.Log) {
	s.log.Infof("AddLog, %v\n", entry)
	s.db.AddLog(entry)
}

func (s *loggingStateDB) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	res := s.db.GetLogs(hash, blockHash)
	s.log.Infof("GetLogs, %v, %v, %v\n", hash, blockHash, res)
	return res
}

func (s *loggingStateDB) Finalise(deleteEmptyObjects bool) {
	s.log.Infof("Finalise, %v\n", deleteEmptyObjects)
	s.db.Finalise(deleteEmptyObjects)
}

func (s *loggingStateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	res := s.db.IntermediateRoot(deleteEmptyObjects)
	s.log.Infof("IntermediateRoot, %v, %v\n", deleteEmptyObjects, res)
	return res
}

func (s *loggingStateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	hash, err := s.db.Commit(deleteEmptyObjects)
	s.log.Infof("Commit, %v, %v, %v\n", deleteEmptyObjects, hash, err)
	return hash, err
}

func (s *loggingStateDB) Prepare(thash common.Hash, ti int) {
	s.db.Prepare(thash, ti)
	s.log.Infof("Prepare, %v, %v\n", thash, ti)
}

func (s *loggingStateDB) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	s.db.PrepareSubstate(substate, block)
	s.log.Infof("PrepareSubstate, %v\n", substate)
}

func (s *loggingStateDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	res := s.db.GetSubstatePostAlloc()
	s.log.Infof("GetSubstatePostAlloc, %v\n", res)
	return res
}

func (s *loggingStateDB) AddPreimage(hash common.Hash, data []byte) {
	s.db.AddPreimage(hash, data)
	s.log.Infof("AddPreimage, %v, %v\n", hash, data)
}

func (s *loggingStateDB) ForEachStorage(addr common.Address, op func(common.Hash, common.Hash) bool) error {
	// no loggin in this case
	return s.db.ForEachStorage(addr, op)
}

func (s *loggingStateDB) StartBulkLoad() BulkLoad {
	// no loggin in this case
	return s.db.StartBulkLoad()
}

func (s *loggingStateDB) GetArchiveState(block uint64) (StateDB, error) {
	// no loggin in this case
	return s.db.GetArchiveState(block)
}

func (s *loggingStateDB) GetMemoryUsage() *MemoryUsage {
	// no loggin in this case
	return s.db.GetMemoryUsage()
}

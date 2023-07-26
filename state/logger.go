package state

import (
	"encoding/hex"
	"math/big"

	"github.com/Fantom-foundation/Aida/logger"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/op/go-logging"
)

// MakeLoggingStateDB wrapps the given StateDB instance into a logging wrapper causing
// every StateDB operation (except BulkLoading) to be logged for debugging.
func MakeLoggingStateDB(db StateDB, logLevel string) StateDB {
	return &loggingStateDB{
		db:  db,
		log: logger.NewLogger(logLevel, "Logging State DB"),
	}
}

type loggingStateDB struct {
	db  StateDB
	log *logging.Logger
}

func (s *loggingStateDB) CreateAccount(addr common.Address) {
	s.db.CreateAccount(addr)
	s.log.Infof("CreateAccount, %v", addr)
}

func (s *loggingStateDB) Exist(addr common.Address) bool {
	res := s.db.Exist(addr)
	s.log.Infof("Exist, %v, %v", addr, res)
	return res
}

func (s *loggingStateDB) Empty(addr common.Address) bool {
	res := s.db.Empty(addr)
	s.log.Infof("Empty, %v, %v", addr, res)
	return res
}

func (s *loggingStateDB) Suicide(addr common.Address) bool {
	res := s.db.Suicide(addr)
	s.log.Infof("Suicide, %v, %v", addr, res)
	return res
}

func (s *loggingStateDB) HasSuicided(addr common.Address) bool {
	res := s.db.HasSuicided(addr)
	s.log.Infof("HasSuicided, %v, %v", addr, res)
	return res
}

func (s *loggingStateDB) GetBalance(addr common.Address) *big.Int {
	res := s.db.GetBalance(addr)
	s.log.Infof("GetBalance, %v, %v", addr, res)
	return res
}

func (s *loggingStateDB) AddBalance(addr common.Address, value *big.Int) {
	s.db.AddBalance(addr, value)
	s.log.Infof("AddBalance, %v, %v, %v", addr, value, s.db.GetBalance(addr))
}

func (s *loggingStateDB) SubBalance(addr common.Address, value *big.Int) {
	s.db.SubBalance(addr, value)
	s.log.Infof("SubBalance, %v, %v, %v", addr, value, s.db.GetBalance(addr))
}

func (s *loggingStateDB) GetNonce(addr common.Address) uint64 {
	res := s.db.GetNonce(addr)
	s.log.Infof("GetNonce, %v, %v", addr, res)
	return res
}

func (s *loggingStateDB) SetNonce(addr common.Address, value uint64) {
	s.db.SetNonce(addr, value)
	s.log.Infof("SetNonce, %v, %v", addr, value)
}

func (s *loggingStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	res := s.db.GetCommittedState(addr, key)
	s.log.Infof("GetCommittedState, %v, %v, %v", addr, key, res)
	return res
}

func (s *loggingStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	res := s.db.GetState(addr, key)
	s.log.Infof("GetState, %v, %v, %v", addr, key, res)
	return res
}

func (s *loggingStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.db.SetState(addr, key, value)
	s.log.Infof("SetState, %v, %v, %v", addr, key, value)
}

func (s *loggingStateDB) GetCode(addr common.Address) []byte {
	res := s.db.GetCode(addr)
	s.log.Infof("GetCode, %v, %v", addr, hex.EncodeToString(res))
	return res
}

func (s *loggingStateDB) GetCodeSize(addr common.Address) int {
	res := s.db.GetCodeSize(addr)
	s.log.Infof("GetCodeSize, %v, %v", addr, res)
	return res
}

func (s *loggingStateDB) GetCodeHash(addr common.Address) common.Hash {
	res := s.db.GetCodeHash(addr)
	s.log.Infof("GetCodeHash, %v, %v", addr, res)
	return res
}

func (s *loggingStateDB) SetCode(addr common.Address, code []byte) {
	s.db.SetCode(addr, code)
	s.log.Infof("SetCode, %v, %v", addr, code)
}

func (s *loggingStateDB) Snapshot() int {
	res := s.db.Snapshot()
	s.log.Infof("Snapshot, %v", res)
	return res
}

func (s *loggingStateDB) RevertToSnapshot(id int) {
	s.db.RevertToSnapshot(id)
	s.log.Infof("RevertToSnapshot, %v", id)
}

func (s *loggingStateDB) Error() error {
	s.log.Error("Error")
	return s.db.Error()
}

func (s *loggingStateDB) BeginTransaction(tx uint32) {
	s.log.Infof("BeginTransaction, %v", tx)
	s.db.BeginTransaction(tx)
}

func (s *loggingStateDB) EndTransaction() {
	s.log.Info("EndTransaction")
	s.db.EndTransaction()
}

func (s *loggingStateDB) BeginBlock(blk uint64) {
	s.log.Infof("BeginBlock, %v", blk)
	s.db.BeginBlock(blk)
}

func (s *loggingStateDB) EndBlock() {
	s.log.Info("EndBlock")
	s.db.EndBlock()
}

func (s *loggingStateDB) BeginSyncPeriod(number uint64) {
	s.log.Infof("BeginSyncPeriod, %v", number)
	s.db.BeginSyncPeriod(number)
}

func (s *loggingStateDB) EndSyncPeriod() {
	s.log.Info("EndSyncPeriod")
	s.db.EndSyncPeriod()
}

func (s *loggingStateDB) Close() error {
	res := s.db.Close()
	s.log.Infof("EndSyncPeriod, %v", res)
	return res
}

func (s *loggingStateDB) AddRefund(amount uint64) {
	s.db.AddRefund(amount)
	s.log.Infof("AddRefund, %v, %v", amount, s.db.GetRefund())
}

func (s *loggingStateDB) SubRefund(amount uint64) {
	s.db.SubRefund(amount)
	s.log.Infof("SubRefund, %v, %v", amount, s.db.GetRefund())
}

func (s *loggingStateDB) GetRefund() uint64 {
	res := s.db.GetRefund()
	s.log.Infof("GetRefund, %v", res)
	return res
}

func (s *loggingStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.log.Infof("PrepareAccessList, %v, %v, %v, %v", sender, dest, precompiles, txAccesses)
	s.db.PrepareAccessList(sender, dest, precompiles, txAccesses)
}

func (s *loggingStateDB) AddressInAccessList(addr common.Address) bool {
	res := s.db.AddressInAccessList(addr)
	s.log.Infof("AddressInAccessList, %v, %v", addr, res)
	return res
}

func (s *loggingStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	a, b := s.db.SlotInAccessList(addr, slot)
	s.log.Infof("SlotInAccessList, %v, %v, %v, %v", addr, slot, a, b)
	return a, b
}

func (s *loggingStateDB) AddAddressToAccessList(addr common.Address) {
	s.log.Infof("AddAddressToAccessList, %v", addr)
	s.db.AddAddressToAccessList(addr)
}

func (s *loggingStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.log.Infof("AddSlotToAccessList, %v, %v", addr, slot)
	s.db.AddSlotToAccessList(addr, slot)
}

func (s *loggingStateDB) AddLog(entry *types.Log) {
	s.log.Infof("AddLog, %v", entry)
	s.db.AddLog(entry)
}

func (s *loggingStateDB) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	res := s.db.GetLogs(hash, blockHash)
	s.log.Infof("GetLogs, %v, %v, %v", hash, blockHash, res)
	return res
}

func (s *loggingStateDB) Finalise(deleteEmptyObjects bool) {
	s.log.Infof("Finalise, %v", deleteEmptyObjects)
	s.db.Finalise(deleteEmptyObjects)
}

func (s *loggingStateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	res := s.db.IntermediateRoot(deleteEmptyObjects)
	s.log.Infof("IntermediateRoot, %v, %v", deleteEmptyObjects, res)
	return res
}

func (s *loggingStateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	hash, err := s.db.Commit(deleteEmptyObjects)
	s.log.Infof("Commit, %v, %v, %v", deleteEmptyObjects, hash, err)
	return hash, err
}

func (s *loggingStateDB) Prepare(thash common.Hash, ti int) {
	s.db.Prepare(thash, ti)
	s.log.Infof("Prepare, %v, %v", thash, ti)
}

func (s *loggingStateDB) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	s.db.PrepareSubstate(substate, block)
	s.log.Infof("PrepareSubstate, %v", substate)
}

func (s *loggingStateDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	res := s.db.GetSubstatePostAlloc()
	s.log.Infof("GetSubstatePostAlloc, %v", res)
	return res
}

func (s *loggingStateDB) AddPreimage(hash common.Hash, data []byte) {
	s.db.AddPreimage(hash, data)
	s.log.Infof("AddPreimage, %v, %v", hash, data)
}

func (s *loggingStateDB) ForEachStorage(addr common.Address, op func(common.Hash, common.Hash) bool) error {
	// no loggin in this case
	return s.db.ForEachStorage(addr, op)
}

func (s *loggingStateDB) StartBulkLoad(block uint64) BulkLoad {
	return &loggingBulkLoad{
		nested: s.db.StartBulkLoad(block),
		log:    s.log,
	}
}

func (s *loggingStateDB) GetArchiveState(block uint64) (StateDB, error) {
	archive, err := s.db.GetArchiveState(block)
	if err != nil {
		return nil, err
	}
	return MakeLoggingStateDB(archive, "DEBUG"), nil
}

func (s *loggingStateDB) GetMemoryUsage() *MemoryUsage {
	// no loggin in this case
	return s.db.GetMemoryUsage()
}

func (s *loggingStateDB) GetShadowDB() StateDB {
	return s.db.GetShadowDB()
}

type loggingBulkLoad struct {
	nested BulkLoad
	log    *logging.Logger
}

func (l *loggingBulkLoad) CreateAccount(addr common.Address) {
	l.nested.CreateAccount(addr)
	l.log.Infof("Bulk, CreateAccount, %v", addr)
}
func (l *loggingBulkLoad) SetBalance(addr common.Address, balance *big.Int) {
	l.nested.SetBalance(addr, balance)
	l.log.Infof("Bulk, SetBalance, %v, %v", addr, balance)
}

func (l *loggingBulkLoad) SetNonce(addr common.Address, nonce uint64) {
	l.nested.SetNonce(addr, nonce)
	l.log.Infof("Bulk, SetNonce, %v, %v", addr, nonce)
}

func (l *loggingBulkLoad) SetState(addr common.Address, key common.Hash, value common.Hash) {
	l.nested.SetState(addr, key, value)
	l.log.Infof("Bulk, SetState, %v, %v, %v", addr, key, value)
}

func (l *loggingBulkLoad) SetCode(addr common.Address, code []byte) {
	l.nested.SetCode(addr, code)
	l.log.Infof("Bulk, SetCode, %v, %v", addr, code)
}

func (l *loggingBulkLoad) Close() error {
	res := l.nested.Close()
	l.log.Infof("Bulk, Close, %v", res)
	return res
}

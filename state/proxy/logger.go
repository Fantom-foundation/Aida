package proxy

import (
	"encoding/hex"
	"math/big"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// NewLoggerProxy wraps the given StateDB instance into a logging wrapper causing
// every StateDB operation (except BulkLoading) to be logged for debugging.
func NewLoggerProxy(db state.StateDB, logLevel string) state.StateDB {
	return &loggingStateDb{
		loggingVmStateDb: loggingVmStateDb{
			db:  db,
			log: logger.NewLogger(logLevel, "Logging State DB"),
		},
		state: db,
	}
}

type loggingVmStateDb struct {
	db  state.VmStateDB
	log logger.Logger
}

type loggingNonCommittableStateDb struct {
	loggingVmStateDb
	nonCommittableStateDB state.NonCommittableStateDB
}

type loggingStateDb struct {
	loggingVmStateDb
	state state.StateDB
}

func (s *loggingVmStateDb) CreateAccount(addr common.Address) {
	s.db.CreateAccount(addr)
	s.log.Infof("CreateAccount, %v", addr)
}

func (s *loggingVmStateDb) Exist(addr common.Address) bool {
	res := s.db.Exist(addr)
	s.log.Infof("Exist, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) Empty(addr common.Address) bool {
	res := s.db.Empty(addr)
	s.log.Infof("Empty, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) Suicide(addr common.Address) bool {
	res := s.db.Suicide(addr)
	s.log.Infof("Suicide, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) HasSuicided(addr common.Address) bool {
	res := s.db.HasSuicided(addr)
	s.log.Infof("HasSuicided, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) GetBalance(addr common.Address) *big.Int {
	res := s.db.GetBalance(addr)
	s.log.Infof("GetBalance, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) AddBalance(addr common.Address, value *big.Int) {
	s.db.AddBalance(addr, value)
	s.log.Infof("AddBalance, %v, %v, %v", addr, value, s.db.GetBalance(addr))
}

func (s *loggingVmStateDb) SubBalance(addr common.Address, value *big.Int) {
	s.db.SubBalance(addr, value)
	s.log.Infof("SubBalance, %v, %v, %v", addr, value, s.db.GetBalance(addr))
}

func (s *loggingVmStateDb) GetNonce(addr common.Address) uint64 {
	res := s.db.GetNonce(addr)
	s.log.Infof("GetNonce, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) SetNonce(addr common.Address, value uint64) {
	s.db.SetNonce(addr, value)
	s.log.Infof("SetNonce, %v, %v", addr, value)
}

func (s *loggingVmStateDb) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	res := s.db.GetCommittedState(addr, key)
	s.log.Infof("GetCommittedState, %v, %v, %v", addr, key, res)
	return res
}

func (s *loggingVmStateDb) GetState(addr common.Address, key common.Hash) common.Hash {
	res := s.db.GetState(addr, key)
	s.log.Infof("GetState, %v, %v, %v", addr, key, res)
	return res
}

func (s *loggingVmStateDb) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.db.SetState(addr, key, value)
	s.log.Infof("SetState, %v, %v, %v", addr, key, value)
}

func (s *loggingVmStateDb) GetCode(addr common.Address) []byte {
	res := s.db.GetCode(addr)
	s.log.Infof("GetCode, %v, %v", addr, hex.EncodeToString(res))
	return res
}

func (s *loggingVmStateDb) GetCodeSize(addr common.Address) int {
	res := s.db.GetCodeSize(addr)
	s.log.Infof("GetCodeSize, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) GetCodeHash(addr common.Address) common.Hash {
	res := s.db.GetCodeHash(addr)
	s.log.Infof("GetCodeHash, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) SetCode(addr common.Address, code []byte) {
	s.db.SetCode(addr, code)
	s.log.Infof("SetCode, %v, %v", addr, code)
}

func (s *loggingVmStateDb) Snapshot() int {
	res := s.db.Snapshot()
	s.log.Infof("Snapshot, %v", res)
	return res
}

func (s *loggingVmStateDb) RevertToSnapshot(id int) {
	s.db.RevertToSnapshot(id)
	s.log.Infof("RevertToSnapshot, %v", id)
}

func (s *loggingStateDb) Error() error {
	s.log.Error("Error")
	return s.state.Error()
}

func (s *loggingVmStateDb) BeginTransaction(tx uint32) {
	s.log.Infof("BeginTransaction, %v", tx)
	s.db.BeginTransaction(tx)
}

func (s *loggingVmStateDb) EndTransaction() {
	s.log.Info("EndTransaction")
	s.db.EndTransaction()
}

func (s *loggingStateDb) BeginBlock(blk uint64) {
	s.log.Infof("BeginBlock, %v", blk)
	s.state.BeginBlock(blk)
}

func (s *loggingStateDb) EndBlock() {
	s.log.Info("EndBlock")
	s.state.EndBlock()
}

func (s *loggingStateDb) BeginSyncPeriod(number uint64) {
	s.log.Infof("BeginSyncPeriod, %v", number)
	s.state.BeginSyncPeriod(number)
}

func (s *loggingStateDb) EndSyncPeriod() {
	s.log.Info("EndSyncPeriod")
	s.state.EndSyncPeriod()
}

func (s *loggingStateDb) GetHash() common.Hash {
	hash := s.state.GetHash()
	s.log.Info("GetHash, %v", hash)
	return hash
}

func (s *loggingNonCommittableStateDb) GetHash() common.Hash {
	hash := s.nonCommittableStateDB.GetHash()
	s.log.Info("GetHash, %v", hash)
	return hash
}

func (s *loggingStateDb) Close() error {
	res := s.state.Close()
	s.log.Infof("EndSyncPeriod, %v", res)
	return res
}

func (s *loggingVmStateDb) AddRefund(amount uint64) {
	s.db.AddRefund(amount)
	s.log.Infof("AddRefund, %v, %v", amount, s.db.GetRefund())
}

func (s *loggingVmStateDb) SubRefund(amount uint64) {
	s.db.SubRefund(amount)
	s.log.Infof("SubRefund, %v, %v", amount, s.db.GetRefund())
}

func (s *loggingVmStateDb) GetRefund() uint64 {
	res := s.db.GetRefund()
	s.log.Infof("GetRefund, %v", res)
	return res
}

func (s *loggingVmStateDb) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.log.Infof("PrepareAccessList, %v, %v, %v, %v", sender, dest, precompiles, txAccesses)
	s.db.PrepareAccessList(sender, dest, precompiles, txAccesses)
}

func (s *loggingVmStateDb) AddressInAccessList(addr common.Address) bool {
	res := s.db.AddressInAccessList(addr)
	s.log.Infof("AddressInAccessList, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	a, b := s.db.SlotInAccessList(addr, slot)
	s.log.Infof("SlotInAccessList, %v, %v, %v, %v", addr, slot, a, b)
	return a, b
}

func (s *loggingVmStateDb) AddAddressToAccessList(addr common.Address) {
	s.log.Infof("AddAddressToAccessList, %v", addr)
	s.db.AddAddressToAccessList(addr)
}

func (s *loggingVmStateDb) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.log.Infof("AddSlotToAccessList, %v, %v", addr, slot)
	s.db.AddSlotToAccessList(addr, slot)
}

func (s *loggingVmStateDb) AddLog(entry *types.Log) {
	s.log.Infof("AddLog, %v", entry)
	s.db.AddLog(entry)
}

func (s *loggingVmStateDb) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	res := s.db.GetLogs(hash, blockHash)
	s.log.Infof("GetLogs, %v, %v, %v", hash, blockHash, res)
	return res
}

func (s *loggingStateDb) Finalise(deleteEmptyObjects bool) {
	s.log.Infof("Finalise, %v", deleteEmptyObjects)
	s.state.Finalise(deleteEmptyObjects)
}

func (s *loggingStateDb) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	res := s.state.IntermediateRoot(deleteEmptyObjects)
	s.log.Infof("IntermediateRoot, %v, %v", deleteEmptyObjects, res)
	return res
}

func (s *loggingStateDb) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	hash, err := s.state.Commit(deleteEmptyObjects)
	s.log.Infof("Commit, %v, %v, %v", deleteEmptyObjects, hash, err)
	return hash, err
}

func (s *loggingVmStateDb) Prepare(thash common.Hash, ti int) {
	s.db.Prepare(thash, ti)
	s.log.Infof("Prepare, %v, %v", thash, ti)
}

func (s *loggingStateDb) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	s.state.PrepareSubstate(substate, block)
	s.log.Infof("PrepareSubstate, %v", substate)
}

func (s *loggingVmStateDb) GetSubstatePostAlloc() substate.SubstateAlloc {
	res := s.db.GetSubstatePostAlloc()
	s.log.Infof("GetSubstatePostAlloc, %v", res)
	return res
}

func (s *loggingVmStateDb) AddPreimage(hash common.Hash, data []byte) {
	s.db.AddPreimage(hash, data)
	s.log.Infof("AddPreimage, %v, %v", hash, data)
}

func (s *loggingVmStateDb) ForEachStorage(addr common.Address, op func(common.Hash, common.Hash) bool) error {
	// no logging in this case
	return s.db.ForEachStorage(addr, op)
}

func (s *loggingStateDb) StartBulkLoad(block uint64) state.BulkLoad {
	return &loggingBulkLoad{
		nested: s.state.StartBulkLoad(block),
		log:    s.log,
	}
}

func (s *loggingStateDb) GetArchiveState(block uint64) (state.NonCommittableStateDB, error) {
	archive, err := s.state.GetArchiveState(block)
	if err != nil {
		return nil, err
	}
	return &loggingNonCommittableStateDb{
		loggingVmStateDb: loggingVmStateDb{
			db:  archive,
			log: logger.NewLogger("DEBUG", "Logging State DB"),
		},
		nonCommittableStateDB: archive,
	}, nil
}

func (s *loggingStateDb) GetArchiveBlockHeight() (uint64, bool, error) {
	res, empty, err := s.state.GetArchiveBlockHeight()
	s.log.Infof("GetArchiveBlockHeight, %v, %t, %v", res, empty, err)
	return res, empty, err
}

func (s *loggingStateDb) GetMemoryUsage() *state.MemoryUsage {
	// no logging in this case
	return s.state.GetMemoryUsage()
}

func (s *loggingStateDb) GetShadowDB() state.StateDB {
	return s.state.GetShadowDB()
}

func (s *loggingNonCommittableStateDb) Release() {
	s.log.Infof("Release")
	s.nonCommittableStateDB.Release()
}

type loggingBulkLoad struct {
	nested state.BulkLoad
	log    logger.Logger
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

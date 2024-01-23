package proxy

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// NewLoggerProxy wraps the given StateDB instance into a logging wrapper causing
// every StateDB operation (except BulkLoading) to be logged for debugging.
func NewLoggerProxy(db state.StateDB, log logger.Logger, output chan string) state.StateDB {
	return &LoggingStateDb{
		loggingVmStateDb: loggingVmStateDb{
			db:     db,
			log:    log,
			output: output,
		},

		state: db,
	}
}

type loggingVmStateDb struct {
	db     state.VmStateDB
	log    logger.Logger
	output chan string
}

type loggingNonCommittableStateDb struct {
	loggingVmStateDb
	nonCommittableStateDB state.NonCommittableStateDB
}

type LoggingStateDb struct {
	loggingVmStateDb
	state state.StateDB
}

func (s *loggingVmStateDb) CreateAccount(addr common.Address) {
	s.db.CreateAccount(addr)
	s.writeLog("CreateAccount, %v", addr)
}

func (s *loggingVmStateDb) Exist(addr common.Address) bool {
	res := s.db.Exist(addr)
	s.writeLog("Exist, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) Empty(addr common.Address) bool {
	res := s.db.Empty(addr)
	s.writeLog("Empty, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) Suicide(addr common.Address) bool {
	res := s.db.Suicide(addr)
	s.writeLog("Suicide, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) HasSuicided(addr common.Address) bool {
	res := s.db.HasSuicided(addr)
	s.writeLog("HasSuicided, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) GetBalance(addr common.Address) *big.Int {
	res := s.db.GetBalance(addr)
	s.writeLog("GetBalance, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) AddBalance(addr common.Address, value *big.Int) {
	s.db.AddBalance(addr, value)
	s.writeLog("AddBalance, %v, %v, %v", addr, value, s.db.GetBalance(addr))
}

func (s *loggingVmStateDb) SubBalance(addr common.Address, value *big.Int) {
	s.db.SubBalance(addr, value)
	s.writeLog("SubBalance, %v, %v, %v", addr, value, s.db.GetBalance(addr))
}

func (s *loggingVmStateDb) GetNonce(addr common.Address) uint64 {
	res := s.db.GetNonce(addr)
	s.writeLog("GetNonce, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) SetNonce(addr common.Address, value uint64) {
	s.db.SetNonce(addr, value)
	s.writeLog("SetNonce, %v, %v", addr, value)
}

func (s *loggingVmStateDb) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	res := s.db.GetCommittedState(addr, key)
	s.writeLog("GetCommittedState, %v, %v, %v", addr, key, res)
	return res
}

func (s *loggingVmStateDb) GetState(addr common.Address, key common.Hash) common.Hash {
	res := s.db.GetState(addr, key)
	s.writeLog("GetState, %v, %v, %v", addr, key, res)
	return res
}

func (s *loggingVmStateDb) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.db.SetState(addr, key, value)
	s.writeLog("SetState, %v, %v, %v", addr, key, value)
}

func (s *loggingVmStateDb) GetCode(addr common.Address) []byte {
	res := s.db.GetCode(addr)
	s.writeLog("GetCode, %v, %v", addr, hex.EncodeToString(res))
	return res
}

func (s *loggingVmStateDb) GetCodeSize(addr common.Address) int {
	res := s.db.GetCodeSize(addr)
	s.writeLog("GetCodeSize, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) GetCodeHash(addr common.Address) common.Hash {
	res := s.db.GetCodeHash(addr)
	s.writeLog("GetCodeHash, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) SetCode(addr common.Address, code []byte) {
	s.db.SetCode(addr, code)
	s.writeLog("SetCode, %v, %v", addr, code)
}

func (s *loggingVmStateDb) Snapshot() int {
	res := s.db.Snapshot()
	s.writeLog("Snapshot, %v", res)
	return res
}

func (s *loggingVmStateDb) RevertToSnapshot(id int) {
	s.db.RevertToSnapshot(id)
	s.writeLog("RevertToSnapshot, %v", id)
}

func (s *LoggingStateDb) Error() error {
	err := s.state.Error()
	s.writeLog("Error, %v", err)
	return err
}

func (s *loggingVmStateDb) BeginTransaction(tx uint32) {
	s.writeLog("BeginTransaction, %v", tx)
	s.db.BeginTransaction(tx)
}

func (s *loggingVmStateDb) EndTransaction() {
	s.writeLog("EndTransaction")
	s.db.EndTransaction()
}

func (s *LoggingStateDb) BeginBlock(blk uint64) {
	s.writeLog("BeginBlock, %v", blk)
	s.state.BeginBlock(blk)
}

func (s *LoggingStateDb) EndBlock() {
	s.writeLog("EndBlock")
	s.state.EndBlock()
}

func (s *LoggingStateDb) BeginSyncPeriod(number uint64) {
	s.writeLog("BeginSyncPeriod, %v", number)
	s.state.BeginSyncPeriod(number)
}

func (s *LoggingStateDb) EndSyncPeriod() {
	s.writeLog("EndSyncPeriod")
	s.state.EndSyncPeriod()
}

func (s *LoggingStateDb) GetHash() common.Hash {
	hash := s.state.GetHash()
	s.writeLog("GetHash, %v", hash)
	return hash
}

func (s *loggingNonCommittableStateDb) GetHash() common.Hash {
	hash := s.nonCommittableStateDB.GetHash()
	s.writeLog("GetHash, %v", hash)
	return hash
}

func (s *LoggingStateDb) Close() error {
	res := s.state.Close()
	str := fmt.Sprintf("EndSyncPeriod, %v", res)
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) AddRefund(amount uint64) {
	s.db.AddRefund(amount)
	s.writeLog("AddRefund, %v, %v", amount, s.db.GetRefund())
}

func (s *loggingVmStateDb) SubRefund(amount uint64) {
	s.db.SubRefund(amount)
	s.writeLog("SubRefund, %v, %v", amount, s.db.GetRefund())
}

func (s *loggingVmStateDb) GetRefund() uint64 {
	res := s.db.GetRefund()
	s.writeLog("GetRefund, %v", res)
	return res
}

func (s *loggingVmStateDb) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.writeLog("PrepareAccessList, %v, %v, %v, %v", sender, dest, precompiles, txAccesses)
	s.db.PrepareAccessList(sender, dest, precompiles, txAccesses)
}

func (s *loggingVmStateDb) AddressInAccessList(addr common.Address) bool {
	res := s.db.AddressInAccessList(addr)
	s.writeLog("AddressInAccessList, %v, %v", addr, res)
	return res
}

func (s *loggingVmStateDb) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	a, b := s.db.SlotInAccessList(addr, slot)
	s.writeLog("SlotInAccessList, %v, %v, %v, %v", addr, slot, a, b)
	return a, b
}

func (s *loggingVmStateDb) AddAddressToAccessList(addr common.Address) {
	s.writeLog("AddAddressToAccessList, %v", addr)
	s.db.AddAddressToAccessList(addr)
}

func (s *loggingVmStateDb) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.writeLog("AddSlotToAccessList, %v, %v", addr, slot)
	s.db.AddSlotToAccessList(addr, slot)
}

func (s *loggingVmStateDb) AddLog(entry *types.Log) {
	s.writeLog("AddLog, %v", entry)
	s.db.AddLog(entry)
}

func (s *loggingVmStateDb) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	res := s.db.GetLogs(hash, blockHash)
	s.writeLog("GetLogs, %v, %v, %v", hash, blockHash, res)
	return res
}

func (s *LoggingStateDb) Finalise(deleteEmptyObjects bool) {
	s.writeLog("Finalise, %v", deleteEmptyObjects)
	s.state.Finalise(deleteEmptyObjects)
}

func (s *LoggingStateDb) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	res := s.state.IntermediateRoot(deleteEmptyObjects)
	s.writeLog("IntermediateRoot, %v, %v", deleteEmptyObjects, res)
	return res
}

func (s *LoggingStateDb) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	hash, err := s.state.Commit(deleteEmptyObjects)
	s.writeLog("Commit, %v, %v, %v", deleteEmptyObjects, hash, err)
	return hash, err
}

func (s *loggingVmStateDb) Prepare(thash common.Hash, ti int) {
	s.db.Prepare(thash, ti)
	s.writeLog("Prepare, %v, %v", thash, ti)
}

func (s *LoggingStateDb) PrepareSubstate(substate txcontext.WorldState, block uint64) {
	s.state.PrepareSubstate(substate, block)
	s.writeLog("PrepareSubstate, %v", substate.String())
}

func (s *loggingVmStateDb) GetSubstatePostAlloc() txcontext.WorldState {
	res := s.db.GetSubstatePostAlloc()
	s.writeLog("GetSubstatePostAlloc, %v", res.String())
	return res
}

func (s *loggingVmStateDb) AddPreimage(hash common.Hash, data []byte) {
	s.db.AddPreimage(hash, data)
	s.writeLog("AddPreimage, %v, %v", hash, data)
}

func (s *loggingVmStateDb) ForEachStorage(addr common.Address, op func(common.Hash, common.Hash) bool) error {
	// no logging in this case
	return s.db.ForEachStorage(addr, op)
}

func (s *LoggingStateDb) StartBulkLoad(block uint64) state.BulkLoad {
	return &loggingBulkLoad{
		nested:   s.state.StartBulkLoad(block),
		writeLog: s.writeLog,
	}
}

func (s *LoggingStateDb) GetArchiveState(block uint64) (state.NonCommittableStateDB, error) {
	archive, err := s.state.GetArchiveState(block)
	if err != nil {
		return nil, err
	}
	return &loggingNonCommittableStateDb{
		loggingVmStateDb: loggingVmStateDb{
			db:     archive,
			log:    s.log,
			output: s.output,
		},
		nonCommittableStateDB: archive,
	}, nil
}

func (s *LoggingStateDb) GetArchiveBlockHeight() (uint64, bool, error) {
	res, empty, err := s.state.GetArchiveBlockHeight()
	s.writeLog("GetArchiveBlockHeight, %v, %t, %v", res, empty, err)
	return res, empty, err
}

func (s *LoggingStateDb) GetMemoryUsage() *state.MemoryUsage {
	// no logging in this case
	return s.state.GetMemoryUsage()
}

func (s *LoggingStateDb) GetShadowDB() state.StateDB {
	return s.state.GetShadowDB()
}

func (s *loggingNonCommittableStateDb) Release() {
	s.writeLog("Release")
	s.nonCommittableStateDB.Release()
}

type loggingBulkLoad struct {
	nested   state.BulkLoad
	writeLog func(format string, a ...any)
}

func (l *loggingBulkLoad) CreateAccount(addr common.Address) {
	l.nested.CreateAccount(addr)
	l.writeLog("Bulk, CreateAccount, %v", addr)
}
func (l *loggingBulkLoad) SetBalance(addr common.Address, balance *big.Int) {
	l.nested.SetBalance(addr, balance)
	l.writeLog("Bulk, SetBalance, %v, %v", addr, balance)
}

func (l *loggingBulkLoad) SetNonce(addr common.Address, nonce uint64) {
	l.nested.SetNonce(addr, nonce)
	l.writeLog("Bulk, SetNonce, %v, %v", addr, nonce)
}

func (l *loggingBulkLoad) SetState(addr common.Address, key common.Hash, value common.Hash) {
	l.nested.SetState(addr, key, value)
	l.writeLog("Bulk, SetState, %v, %v, %v", addr, key, value)
}

func (l *loggingBulkLoad) SetCode(addr common.Address, code []byte) {
	l.nested.SetCode(addr, code)
	l.writeLog("Bulk, SetCode, %v, %v", addr, code)
}

func (l *loggingBulkLoad) Close() error {
	res := l.nested.Close()
	l.writeLog("Bulk, Close, %v", res)
	return res
}

func (s *loggingVmStateDb) writeLog(format string, a ...any) {
	str := fmt.Sprintf(format, a...)
	s.output <- str
	s.log.Debug(str)
}

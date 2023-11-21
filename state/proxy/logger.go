package proxy

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// NewLoggerProxy wraps the given StateDB instance into a logging wrapper causing
// every StateDB operation (except BulkLoading) to be logged for debugging.
func NewLoggerProxy(db state.StateDB, log logger.Logger, output chan string) state.StateDB {
	return &loggingStateDb{
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

type loggingStateDb struct {
	loggingVmStateDb
	state state.StateDB
}

func (s *loggingVmStateDb) CreateAccount(addr common.Address) {
	s.db.CreateAccount(addr)
	str := fmt.Sprintf("CreateAccount, %v", addr)
	s.output <- str
	s.log.Debug(str)
}

func (s *loggingVmStateDb) Exist(addr common.Address) bool {
	res := s.db.Exist(addr)
	str := fmt.Sprintf("Exist, %v, %v", addr, res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) Empty(addr common.Address) bool {
	res := s.db.Empty(addr)
	str := fmt.Sprintf("Empty, %v, %v", addr, res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) Suicide(addr common.Address) bool {
	res := s.db.Suicide(addr)
	str := fmt.Sprintf("Suicide, %v, %v", addr, res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) HasSuicided(addr common.Address) bool {
	res := s.db.HasSuicided(addr)
	str := fmt.Sprintf("HasSuicided, %v, %v", addr, res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) GetBalance(addr common.Address) *big.Int {
	res := s.db.GetBalance(addr)
	str := fmt.Sprintf("GetBalance, %v, %v", addr, res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) AddBalance(addr common.Address, value *big.Int) {
	s.db.AddBalance(addr, value)
	str := fmt.Sprintf("AddBalance, %v, %v, %v", addr, value, s.db.GetBalance(addr))
	s.output <- str
	s.log.Debug(str)
}

func (s *loggingVmStateDb) SubBalance(addr common.Address, value *big.Int) {
	s.db.SubBalance(addr, value)
	str := fmt.Sprintf("SubBalance, %v, %v, %v", addr, value, s.db.GetBalance(addr))
	s.output <- str
	s.log.Debug(str)
}

func (s *loggingVmStateDb) GetNonce(addr common.Address) uint64 {
	res := s.db.GetNonce(addr)
	str := fmt.Sprintf("GetNonce, %v, %v", addr, res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) SetNonce(addr common.Address, value uint64) {
	s.db.SetNonce(addr, value)
	str := fmt.Sprintf("SetNonce, %v, %v", addr, value)
	s.output <- str
	s.log.Debug(str)
}

func (s *loggingVmStateDb) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	res := s.db.GetCommittedState(addr, key)
	str := fmt.Sprintf("GetCommittedState, %v, %v, %v", addr, key, res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) GetState(addr common.Address, key common.Hash) common.Hash {
	res := s.db.GetState(addr, key)
	str := fmt.Sprintf("GetState, %v, %v, %v", addr, key, res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.db.SetState(addr, key, value)
	str := fmt.Sprintf("SetState, %v, %v, %v", addr, key, value)
	s.output <- str
	s.log.Debug(str)
}

func (s *loggingVmStateDb) GetCode(addr common.Address) []byte {
	res := s.db.GetCode(addr)
	str := fmt.Sprintf("GetCode, %v, %v", addr, hex.EncodeToString(res))
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) GetCodeSize(addr common.Address) int {
	res := s.db.GetCodeSize(addr)
	str := fmt.Sprintf("GetCodeSize, %v, %v", addr, res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) GetCodeHash(addr common.Address) common.Hash {
	res := s.db.GetCodeHash(addr)
	str := fmt.Sprintf("GetCodeHash, %v, %v", addr, res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) SetCode(addr common.Address, code []byte) {
	s.db.SetCode(addr, code)
	str := fmt.Sprintf("SetCode, %v, %v", addr, code)
	s.output <- str
	s.log.Debug(str)
}

func (s *loggingVmStateDb) Snapshot() int {
	res := s.db.Snapshot()
	str := fmt.Sprintf("Snapshot, %v", res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) RevertToSnapshot(id int) {
	s.db.RevertToSnapshot(id)
	str := fmt.Sprintf("RevertToSnapshot, %v", id)
	s.output <- str
	s.log.Debug(str)
}

func (s *loggingStateDb) Error() error {
	str := "Error"
	s.log.Debug(str)
	s.output <- str
	return s.state.Error()
}

func (s *loggingVmStateDb) BeginTransaction(tx uint32) {
	str := fmt.Sprintf("BeginTransaction, %v", tx)
	s.output <- str
	s.log.Debug(str)
	s.db.BeginTransaction(tx)
}

func (s *loggingVmStateDb) EndTransaction() {
	str := "EndTransaction"
	s.output <- str
	s.log.Debug(str)
	s.db.EndTransaction()
}

func (s *loggingStateDb) BeginBlock(blk uint64) {
	str := fmt.Sprintf("BeginBlock, %v", blk)
	s.output <- str
	s.log.Debug(str)
	s.state.BeginBlock(blk)
}

func (s *loggingStateDb) EndBlock() {
	str := "EndBlock"
	s.output <- str
	s.log.Debug(str)
	s.state.EndBlock()
}

func (s *loggingStateDb) BeginSyncPeriod(number uint64) {
	str := fmt.Sprintf("BeginSyncPeriod, %v", number)
	s.output <- str
	s.log.Debug(str)
	s.state.BeginSyncPeriod(number)
}

func (s *loggingStateDb) EndSyncPeriod() {
	str := "EndSyncPeriod"
	s.output <- str
	s.log.Debug(str)
	s.state.EndSyncPeriod()
}

func (s *loggingStateDb) GetHash() common.Hash {
	hash := s.state.GetHash()
	str := fmt.Sprintf("GetHash, %v", hash)
	s.output <- str
	s.log.Debug(str)
	return hash
}

func (s *loggingNonCommittableStateDb) GetHash() common.Hash {
	hash := s.nonCommittableStateDB.GetHash()
	str := fmt.Sprintf("GetHash, %v", hash)
	s.output <- str
	s.log.Debug(str)
	return hash
}

func (s *loggingStateDb) Close() error {
	res := s.state.Close()
	str := fmt.Sprintf("EndSyncPeriod, %v", res)
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) AddRefund(amount uint64) {
	s.db.AddRefund(amount)
	str := fmt.Sprintf("AddRefund, %v, %v", amount, s.db.GetRefund())
	s.output <- str
	s.log.Debug(str)
}

func (s *loggingVmStateDb) SubRefund(amount uint64) {
	s.db.SubRefund(amount)
	str := fmt.Sprintf("SubRefund, %v, %v", amount, s.db.GetRefund())
	s.output <- str
	s.log.Debug(str)
}

func (s *loggingVmStateDb) GetRefund() uint64 {
	res := s.db.GetRefund()
	str := fmt.Sprintf("GetRefund, %v", res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	str := fmt.Sprintf("PrepareAccessList, %v, %v, %v, %v", sender, dest, precompiles, txAccesses)
	s.output <- str
	s.log.Debug(str)
	s.db.PrepareAccessList(sender, dest, precompiles, txAccesses)
}

func (s *loggingVmStateDb) AddressInAccessList(addr common.Address) bool {
	res := s.db.AddressInAccessList(addr)
	str := fmt.Sprintf("AddressInAccessList, %v, %v", addr, res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	a, b := s.db.SlotInAccessList(addr, slot)
	str := fmt.Sprintf("SlotInAccessList, %v, %v, %v, %v", addr, slot, a, b)
	s.output <- str
	s.log.Debug(str)
	return a, b
}

func (s *loggingVmStateDb) AddAddressToAccessList(addr common.Address) {
	str := fmt.Sprintf("AddAddressToAccessList, %v", addr)
	s.output <- str
	s.log.Debug(str)
	s.db.AddAddressToAccessList(addr)
}

func (s *loggingVmStateDb) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	str := fmt.Sprintf("AddSlotToAccessList, %v, %v", addr, slot)
	s.output <- str
	s.log.Debug(str)
	s.db.AddSlotToAccessList(addr, slot)
}

func (s *loggingVmStateDb) AddLog(entry *types.Log) {
	str := fmt.Sprintf("AddLog, %v", entry)
	s.output <- str
	s.log.Debug(str)
	s.db.AddLog(entry)
}

func (s *loggingVmStateDb) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	res := s.db.GetLogs(hash, blockHash)
	str := fmt.Sprintf("GetLogs, %v, %v, %v", hash, blockHash, res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingStateDb) Finalise(deleteEmptyObjects bool) {
	str := fmt.Sprintf("Finalise, %v", deleteEmptyObjects)
	s.output <- str
	s.log.Debug(str)
	s.state.Finalise(deleteEmptyObjects)
}

func (s *loggingStateDb) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	res := s.state.IntermediateRoot(deleteEmptyObjects)
	str := fmt.Sprintf("IntermediateRoot, %v, %v", deleteEmptyObjects, res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingStateDb) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	hash, err := s.state.Commit(deleteEmptyObjects)
	str := fmt.Sprintf("Commit, %v, %v, %v", deleteEmptyObjects, hash, err)
	s.output <- str
	s.log.Debug(str)
	return hash, err
}

func (s *loggingVmStateDb) Prepare(thash common.Hash, ti int) {
	s.db.Prepare(thash, ti)
	str := fmt.Sprintf("Prepare, %v, %v", thash, ti)
	s.output <- str
	s.log.Debug(str)
}

func (s *loggingStateDb) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	s.state.PrepareSubstate(substate, block)
	str := fmt.Sprintf("PrepareSubstate, %v", substate)
	s.output <- str
	s.log.Debug(str)
}

func (s *loggingVmStateDb) GetSubstatePostAlloc() substate.SubstateAlloc {
	res := s.db.GetSubstatePostAlloc()
	str := fmt.Sprintf("GetSubstatePostAlloc, %v", res)
	s.output <- str
	s.log.Debug(str)
	return res
}

func (s *loggingVmStateDb) AddPreimage(hash common.Hash, data []byte) {
	s.db.AddPreimage(hash, data)
	str := fmt.Sprintf("AddPreimage, %v, %v", hash, data)
	s.output <- str
	s.log.Debug(str)
}

func (s *loggingVmStateDb) ForEachStorage(addr common.Address, op func(common.Hash, common.Hash) bool) error {
	// no logging in this case
	return s.db.ForEachStorage(addr, op)
}

func (s *loggingStateDb) StartBulkLoad(block uint64) state.BulkLoad {
	return &loggingBulkLoad{
		nested: s.state.StartBulkLoad(block),
		log:    s.log,
		output: s.output,
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
	str := fmt.Sprintf("GetArchiveBlockHeight, %v, %t, %v", res, empty, err)
	s.log.Debug(str)
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
	str := "Release"
	s.log.Debug(str)
	s.nonCommittableStateDB.Release()
}

type loggingBulkLoad struct {
	nested state.BulkLoad
	log    logger.Logger
	output chan string
}

func (l *loggingBulkLoad) CreateAccount(addr common.Address) {
	l.nested.CreateAccount(addr)
	str := fmt.Sprintf("Bulk, CreateAccount, %v", addr)
	l.output <- str
	l.log.Debug(str)
}
func (l *loggingBulkLoad) SetBalance(addr common.Address, balance *big.Int) {
	l.nested.SetBalance(addr, balance)
	str := fmt.Sprintf("Bulk, SetBalance, %v, %v", addr, balance)
	l.output <- str
	l.log.Debug(str)
}

func (l *loggingBulkLoad) SetNonce(addr common.Address, nonce uint64) {
	l.nested.SetNonce(addr, nonce)
	str := fmt.Sprintf("Bulk, SetNonce, %v, %v", addr, nonce)
	l.output <- str
	l.log.Debug(str)
}

func (l *loggingBulkLoad) SetState(addr common.Address, key common.Hash, value common.Hash) {
	l.nested.SetState(addr, key, value)
	str := fmt.Sprintf("Bulk, SetState, %v, %v, %v", addr, key, value)
	l.output <- str
	l.log.Debug(str)
}

func (l *loggingBulkLoad) SetCode(addr common.Address, code []byte) {
	l.nested.SetCode(addr, code)
	str := fmt.Sprintf("Bulk, SetCode, %v, %v", addr, code)
	l.output <- str
	l.log.Debug(str)
}

func (l *loggingBulkLoad) Close() error {
	res := l.nested.Close()
	str := fmt.Sprintf("Bulk, Close, %v", res)
	l.output <- str
	l.log.Debug(str)
	return res
}

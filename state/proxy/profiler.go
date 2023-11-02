package proxy

import (
	"fmt"
	"math/big"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/tracer/profile"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/op/go-logging"
)

// ProfilerProxy data structure for capturing and recording
// invoked StateDB operations.
type ProfilerProxy struct {
	db  state.StateDB  // state db
	ps  *profile.Stats // operation statistics
	log *logging.Logger
}

// NewProfilerProxy creates a new StateDB profiler.
func NewProfilerProxy(db state.StateDB, csv string, logLevel string) (*ProfilerProxy, *profile.Stats) {
	p := new(ProfilerProxy)
	p.db = db
	p.ps = profile.NewStats(csv)
	p.ps.FillLabels(operation.CreateIdLabelMap())
	p.log = logger.NewLogger(logLevel, "Proxy Profiler")
	return p, p.ps
}

// CreateAccount creates a new account.
func (p *ProfilerProxy) CreateAccount(addr common.Address) {
	p.do(operation.CreateAccountID, func() {
		p.db.CreateAccount(addr)
	})
}

// SubBalance subtracts amount from a contract address.
func (p *ProfilerProxy) SubBalance(addr common.Address, amount *big.Int) {
	p.do(operation.SubBalanceID, func() {
		p.db.SubBalance(addr, amount)
	})
}

// AddBalance adds amount to a contract address.
func (p *ProfilerProxy) AddBalance(addr common.Address, amount *big.Int) {
	p.do(operation.AddBalanceID, func() {
		p.db.AddBalance(addr, amount)
	})
}

// GetBalance retrieves the amount of a contract address.
func (p *ProfilerProxy) GetBalance(addr common.Address) *big.Int {
	var res *big.Int
	p.do(operation.GetBalanceID, func() {
		res = p.db.GetBalance(addr)
	})
	return res
}

// GetNonce retrieves the nonce of a contract address.
func (p *ProfilerProxy) GetNonce(addr common.Address) uint64 {
	var res uint64
	p.do(operation.GetNonceID, func() {
		res = p.db.GetNonce(addr)
	})
	return res
}

// SetNonce sets the nonce of a contract address.
func (p *ProfilerProxy) SetNonce(addr common.Address, nonce uint64) {
	p.do(operation.SetNonceID, func() {
		p.db.SetNonce(addr, nonce)
	})
}

// GetCodeHash returns the hash of the EVM bytecode.
func (p *ProfilerProxy) GetCodeHash(addr common.Address) common.Hash {
	var res common.Hash
	p.do(operation.GetCodeHashID, func() {
		res = p.db.GetCodeHash(addr)
	})
	return res
}

// GetCode returns the EVM bytecode of a contract.
func (p *ProfilerProxy) GetCode(addr common.Address) []byte {
	var res []byte
	p.do(operation.GetCodeID, func() {
		res = p.db.GetCode(addr)
	})
	return res
}

// SetCode sets the EVM bytecode of a contract.
func (p *ProfilerProxy) SetCode(addr common.Address, code []byte) {
	p.do(operation.SetCodeID, func() {
		p.db.SetCode(addr, code)
	})
}

// GetCodeSize returns the EVM bytecode's size.
func (p *ProfilerProxy) GetCodeSize(addr common.Address) int {
	var res int
	p.do(operation.GetCodeSizeID, func() {
		res = p.db.GetCodeSize(addr)
	})
	return res
}

// AddRefund adds gas to the refund counter.
func (p *ProfilerProxy) AddRefund(gas uint64) {
	p.do(operation.AddRefundID, func() {
		p.db.AddRefund(gas)
	})
}

// SubRefund subtracts gas to the refund counter.
func (p *ProfilerProxy) SubRefund(gas uint64) {
	p.do(operation.SubRefundID, func() {
		p.db.SubRefund(gas)
	})
}

// GetRefund returns the current value of the refund counter.
func (p *ProfilerProxy) GetRefund() uint64 {
	var res uint64
	p.do(operation.GetRefundID, func() {
		res = p.db.GetRefund()
	})
	return res
}

// GetCommittedState retrieves a value that is already committed.
func (p *ProfilerProxy) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	var res common.Hash
	p.do(operation.GetCommittedStateID, func() {
		res = p.db.GetCommittedState(addr, key)
	})
	return res
}

// GetState retrieves a value from the StateDB.
func (p *ProfilerProxy) GetState(addr common.Address, key common.Hash) common.Hash {
	var res common.Hash
	p.do(operation.GetStateID, func() {
		res = p.db.GetState(addr, key)
	})
	return res
}

// SetState sets a value in the StateDB.
func (p *ProfilerProxy) SetState(addr common.Address, key common.Hash, value common.Hash) {
	p.do(operation.SetStateID, func() {
		p.db.SetState(addr, key, value)
	})
}

// Suicide marks the given account as suicided. This clears the account balance.
// The account is still available until the state is committed;
// return a non-nil account after Suicide.
func (p *ProfilerProxy) Suicide(addr common.Address) bool {
	var suicide bool
	p.do(operation.SuicideID, func() {
		suicide = p.db.Suicide(addr)
	})
	return suicide
}

// HasSuicided checks whether a contract has been suicided.
func (p *ProfilerProxy) HasSuicided(addr common.Address) bool {
	var res bool
	p.do(operation.HasSuicidedID, func() {
		res = p.db.HasSuicided(addr)
	})
	return res
}

// Exist checks whether the contract exists in the StateDB.
// Notably this also returns true for suicided accounts.
func (p *ProfilerProxy) Exist(addr common.Address) bool {
	var res bool
	p.do(operation.ExistID, func() {
		res = p.db.Exist(addr)
	})
	return res
}

// Empty checks whether the contract is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0).
func (p *ProfilerProxy) Empty(addr common.Address) bool {
	var empty bool
	p.do(operation.EmptyID, func() {
		empty = p.db.Empty(addr)
	})
	return empty
}

// PrepareAccessList handles the preparatory steps for executing a state transition with
// regards to both EIP-2929 and EIP-2930:
//
// - Add sender to access list (2929)
// - Add destination to access list (2929)
// - Add precompiles to access list (2929)
// - Add the contents of the optional tx access list (2930)
//
// This method should only be called if Berlin/2929+2930 is applicable at the current number.
func (p *ProfilerProxy) PrepareAccessList(render common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	p.do(operation.PrepareAccessListID, func() {
		p.db.PrepareAccessList(render, dest, precompiles, txAccesses)
	})
}

// AddAddressToAccessList adds an address to the access list.
func (p *ProfilerProxy) AddAddressToAccessList(addr common.Address) {
	p.do(operation.AddAddressToAccessListID, func() {
		p.db.AddAddressToAccessList(addr)
	})
}

// AddressInAccessList checks whether an address is in the access list.
func (p *ProfilerProxy) AddressInAccessList(addr common.Address) bool {
	var res bool
	p.do(operation.AddressInAccessListID, func() {
		res = p.db.AddressInAccessList(addr)
	})
	return res
}

// SlotInAccessList checks whether the (address, slot)-tuple is in the access list.
func (p *ProfilerProxy) SlotInAccessList(addr common.Address, slot common.Hash) (bool, bool) {
	var addressOk, slotOk bool
	p.do(operation.SlotInAccessListID, func() {
		addressOk, slotOk = p.db.SlotInAccessList(addr, slot)
	})
	return addressOk, slotOk
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (p *ProfilerProxy) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	p.do(operation.AddSlotToAccessListID, func() {
		p.db.AddSlotToAccessList(addr, slot)
	})
}

// Snapshot returns an identifier for the current revision of the state.
func (p *ProfilerProxy) Snapshot() int {
	var res int
	p.do(operation.SnapshotID, func() {
		res = p.db.Snapshot()
	})
	return res
}

// RevertToSnapshot reverts all state changes from a given revision.
func (p *ProfilerProxy) RevertToSnapshot(snapshot int) {
	p.do(operation.RevertToSnapshotID, func() {
		p.db.RevertToSnapshot(snapshot)
	})
}

func (p *ProfilerProxy) Error() error {
	return p.db.Error()
}

func (p *ProfilerProxy) do(opId byte, op func()) {
	start := time.Now()
	op()
	elapsed := time.Since(start)
	p.ps.Profile(opId, elapsed)
}

func (p *ProfilerProxy) BeginTransaction(number uint32) {
	p.do(operation.BeginTransactionID, func() {
		p.db.BeginTransaction(number)
	})
}

func (p *ProfilerProxy) EndTransaction() {
	p.do(operation.EndTransactionID, func() {
		p.db.EndTransaction()
	})
}

func (p *ProfilerProxy) BeginBlock(number uint64) {
	p.do(operation.BeginBlockID, func() {
		p.db.BeginBlock(number)
	})
}

func (p *ProfilerProxy) EndBlock() {
	p.do(operation.EndBlockID, func() {
		p.db.EndBlock()
	})
}

func (p *ProfilerProxy) BeginSyncPeriod(number uint64) {
	p.do(operation.BeginSyncPeriodID, func() {
		p.db.BeginSyncPeriod(number)
	})
}

func (p *ProfilerProxy) EndSyncPeriod() {
	p.do(operation.EndSyncPeriodID, func() {
		p.db.EndSyncPeriod()
	})
}

func (p *ProfilerProxy) GetHash() common.Hash {
	// TODO: add profiling for this operation
	return p.db.GetHash()
}

// AddLog adds a log entry.
func (p *ProfilerProxy) AddLog(log *types.Log) {
	p.do(operation.AddLogID, func() {
		p.db.AddLog(log)
	})
}

// GetLogs retrieves log entries.
func (p *ProfilerProxy) GetLogs(hash common.Hash, blockHash common.Hash) (logs []*types.Log) {
	p.do(operation.GetLogsID, func() {
		logs = p.db.GetLogs(hash, blockHash)
	})
	return logs
}

// AddPreimage adds a SHA3 preimage.
func (p *ProfilerProxy) AddPreimage(addr common.Hash, image []byte) {
	p.do(operation.AddPreimageID, func() {
		p.db.AddPreimage(addr, image)
	})
}

// ForEachStorage performs a function over all storage locations in a contract.
func (p *ProfilerProxy) ForEachStorage(addr common.Address, fn func(common.Hash, common.Hash) bool) error {
	var err error
	p.do(operation.ForEachStorageID, func() {
		err = p.db.ForEachStorage(addr, fn)
	})
	return err
}

// Prepare sets the current transaction hash and index.
func (p *ProfilerProxy) Prepare(thash common.Hash, ti int) {
	p.do(operation.PrepareID, func() {
		p.db.Prepare(thash, ti)
	})
}

// Finalise the state in StateDB.
func (p *ProfilerProxy) Finalise(deleteEmptyObjects bool) {
	p.do(operation.FinaliseID, func() {
		p.db.Finalise(deleteEmptyObjects)
	})
}

// IntermediateRoot computes the current hash of the StateDB.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (p *ProfilerProxy) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	var hash common.Hash
	p.do(operation.IntermediateRootID, func() {
		hash = p.db.IntermediateRoot(deleteEmptyObjects)
	})
	return hash
}

func (p *ProfilerProxy) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	var hash common.Hash
	var err error
	p.do(operation.CommitID, func() {
		hash, err = p.db.Commit(deleteEmptyObjects)
	})
	return hash, err
}

// GetSubstatePostAlloc gets substate post allocation.
func (p *ProfilerProxy) GetSubstatePostAlloc() substate.SubstateAlloc {
	return p.db.GetSubstatePostAlloc()
}

func (p *ProfilerProxy) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	p.db.PrepareSubstate(substate, block)
}

func (p *ProfilerProxy) Close() error {
	var err error
	p.do(operation.CloseID, func() {
		err = p.db.Close()
	})
	return err
}

func (p *ProfilerProxy) StartBulkLoad(block uint64) state.BulkLoad {
	p.log.Fatal("StartBulkLoad not supported by ProfilerProxy")
	return nil
}

func (p *ProfilerProxy) GetArchiveState(block uint64) (state.NonCommittableStateDB, error) {
	return nil, fmt.Errorf("archive states are not (yet) supported by the profiling profiler")
}

func (p *ProfilerProxy) GetArchiveBlockHeight() (uint64, bool, error) {
	return 0, false, fmt.Errorf("archive states are not (yet) supported by the profiling profiler")
}

func (p *ProfilerProxy) GetMemoryUsage() *state.MemoryUsage {
	return p.db.GetMemoryUsage()
}

func (p *ProfilerProxy) GetShadowDB() state.StateDB {
	return p.db.GetShadowDB()
}

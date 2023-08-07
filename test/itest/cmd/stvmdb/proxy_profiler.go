package stvmdb

import (
	"fmt"
	"math/big"
	"time"

	"github.com/Fantom-foundation/rc-testing/test/itest/state"
	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/operation"
	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/profile"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ProxyProfiler data structure for capturing and recording
// invoked StateDB operations.
type ProxyProfiler struct {
	db state.StateDB  // state db
	ps *profile.Stats // operation statistics
}

// NewProxyProfiler creates a new StateDB proxy.
func NewProxyProfiler(db state.StateDB, csv string) (*ProxyProfiler, *profile.Stats) {
	p := new(ProxyProfiler)
	p.db = db
	p.ps = profile.NewStats(csv)
	p.ps.FillLabels(operation.CreateIdLabelMap())
	return p, p.ps
}

// CreateAccounts creates a new account.
func (p *ProxyProfiler) CreateAccount(addr common.Address) {
	start := time.Now()
	p.db.CreateAccount(addr)
	elapsed := time.Since(start)
	p.ps.Profile(operation.CreateAccountID, elapsed)
}

// SubtractBalance subtracts amount from a contract address.
func (p *ProxyProfiler) SubBalance(addr common.Address, amount *big.Int) {
	start := time.Now()
	p.db.SubBalance(addr, amount)
	elapsed := time.Since(start)
	p.ps.Profile(operation.SubBalanceID, elapsed)
}

// AddBalance adds amount to a contract address.
func (p *ProxyProfiler) AddBalance(addr common.Address, amount *big.Int) {
	start := time.Now()
	p.db.AddBalance(addr, amount)
	elapsed := time.Since(start)
	p.ps.Profile(operation.AddBalanceID, elapsed)
}

// GetBalance retrieves the amount of a contract address.
func (p *ProxyProfiler) GetBalance(addr common.Address) *big.Int {
	start := time.Now()
	balance := p.db.GetBalance(addr)
	elapsed := time.Since(start)
	p.ps.Profile(operation.GetBalanceID, elapsed)
	return balance
}

// GetNonce retrieves the nonce of a contract address.
func (p *ProxyProfiler) GetNonce(addr common.Address) uint64 {
	start := time.Now()
	nonce := p.db.GetNonce(addr)
	elapsed := time.Since(start)
	p.ps.Profile(operation.GetBalanceID, elapsed)
	return nonce
}

// SetNonce sets the nonce of a contract address.
func (p *ProxyProfiler) SetNonce(addr common.Address, nonce uint64) {
	start := time.Now()
	p.db.SetNonce(addr, nonce)
	elapsed := time.Since(start)
	p.ps.Profile(operation.SetNonceID, elapsed)
}

// GetCodeHash returns the hash of the EVM bytecode.
func (p *ProxyProfiler) GetCodeHash(addr common.Address) common.Hash {
	start := time.Now()
	hash := p.db.GetCodeHash(addr)
	elapsed := time.Since(start)
	p.ps.Profile(operation.GetCodeHashID, elapsed)
	return hash
}

// GetCode returns the EVM bytecode of a contract.
func (p *ProxyProfiler) GetCode(addr common.Address) []byte {
	start := time.Now()
	code := p.db.GetCode(addr)
	elapsed := time.Since(start)
	p.ps.Profile(operation.GetCodeID, elapsed)
	return code
}

// Setcode sets the EVM bytecode of a contract.
func (p *ProxyProfiler) SetCode(addr common.Address, code []byte) {
	start := time.Now()
	p.db.SetCode(addr, code)
	elapsed := time.Since(start)
	p.ps.Profile(operation.SetCodeID, elapsed)
}

// GetCodeSize returns the EVM bytecode's size.
func (p *ProxyProfiler) GetCodeSize(addr common.Address) int {
	start := time.Now()
	size := p.db.GetCodeSize(addr)
	elapsed := time.Since(start)
	p.ps.Profile(operation.GetCodeSizeID, elapsed)
	return size
}

// AddRefund adds gas to the refund counter.
func (p *ProxyProfiler) AddRefund(gas uint64) {
	p.do(operation.AddRefundID, func() {
		p.db.AddRefund(gas)
	})
}

// SubRefund subtracts gas to the refund counter.
func (p *ProxyProfiler) SubRefund(gas uint64) {
	p.do(operation.SubRefundID, func() {
		p.db.SubRefund(gas)
	})
}

// GetRefund returns the current value of the refund counter.
func (p *ProxyProfiler) GetRefund() uint64 {
	var res uint64
	p.do(operation.GetRefundID, func() {
		res = p.db.GetRefund()
	})
	return res
}

// GetCommittedState retrieves a value that is already committed.
func (p *ProxyProfiler) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	start := time.Now()
	value := p.db.GetCommittedState(addr, key)
	elapsed := time.Since(start)
	p.ps.Profile(operation.GetCommittedStateID, elapsed)
	return value
}

// GetState retrieves a value from the StateDB.
func (p *ProxyProfiler) GetState(addr common.Address, key common.Hash) common.Hash {
	start := time.Now()
	value := p.db.GetState(addr, key)
	elapsed := time.Since(start)
	p.ps.Profile(operation.GetStateID, elapsed)
	return value
}

// SetState sets a value in the StateDB.
func (p *ProxyProfiler) SetState(addr common.Address, key common.Hash, value common.Hash) {
	start := time.Now()
	p.db.SetState(addr, key, value)
	elapsed := time.Since(start)
	p.ps.Profile(operation.SetStateID, elapsed)
}

// Suicide marks the given account as suicided. This clears the account balance.
// The account is still available until the state is committed;
// return a non-nil account after Suicide.
func (p *ProxyProfiler) Suicide(addr common.Address) bool {
	start := time.Now()
	suicide := p.db.Suicide(addr)
	elapsed := time.Since(start)
	p.ps.Profile(operation.SuicideID, elapsed)
	return suicide
}

// HasSuicided checks whether a contract has been suicided.
func (p *ProxyProfiler) HasSuicided(addr common.Address) bool {
	var res bool
	p.do(operation.HasSuicidedID, func() {
		res = p.db.HasSuicided(addr)
	})
	return res
}

// Exist checks whether the contract exists in the StateDB.
// Notably this also returns true for suicided accounts.
func (p *ProxyProfiler) Exist(addr common.Address) bool {
	start := time.Now()
	exist := p.db.Exist(addr)
	elapsed := time.Since(start)
	p.ps.Profile(operation.ExistID, elapsed)
	return exist
}

// Empty checks whether the contract is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0).
func (p *ProxyProfiler) Empty(addr common.Address) bool {
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
func (p *ProxyProfiler) PrepareAccessList(render common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	p.do(operation.PrepareAccessListID, func() {
		p.db.PrepareAccessList(render, dest, precompiles, txAccesses)
	})
}

// AddAddressToAccessList adds an address to the access list.
func (p *ProxyProfiler) AddAddressToAccessList(addr common.Address) {
	p.do(operation.AddAddressToAccessListID, func() {
		p.db.AddAddressToAccessList(addr)
	})
}

// AddressInAccessList checks whether an address is in the access list.
func (p *ProxyProfiler) AddressInAccessList(addr common.Address) bool {
	res := false
	p.do(operation.AddressInAccessListID, func() {
		res = p.db.AddressInAccessList(addr)
	})
	return res
}

// SlotInAccessList checks whether the (address, slot)-tuple is in the access list.
func (p *ProxyProfiler) SlotInAccessList(addr common.Address, slot common.Hash) (bool, bool) {
	var addressOk, slotOk bool
	p.do(operation.SlotInAccessListID, func() {
		addressOk, slotOk = p.db.SlotInAccessList(addr, slot)
	})
	return addressOk, slotOk
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (p *ProxyProfiler) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	p.do(operation.AddSlotToAccessListID, func() {
		p.db.AddSlotToAccessList(addr, slot)
	})
}

// Snapshot returns an identifier for the current revision of the state.
func (p *ProxyProfiler) Snapshot() int {
	start := time.Now()
	snapshot := p.db.Snapshot()
	elapsed := time.Since(start)
	p.ps.Profile(operation.SnapshotID, elapsed)
	return snapshot
}

// RevertToSnapshot reverts all state changes from a given revision.
func (p *ProxyProfiler) RevertToSnapshot(snapshot int) {
	start := time.Now()
	p.db.RevertToSnapshot(snapshot)
	elapsed := time.Since(start)
	p.ps.Profile(operation.RevertToSnapshotID, elapsed)
}

func (p *ProxyProfiler) Error() error {
	return p.db.Error()
}

func (p *ProxyProfiler) do(opId byte, op func()) {
	start := time.Now()
	op()
	elapsed := time.Since(start)
	p.ps.Profile(opId, elapsed)
}

func (p *ProxyProfiler) BeginTransaction(number uint32) {
	p.do(operation.BeginTransactionID, func() {
		p.db.BeginTransaction(number)
	})
}

func (p *ProxyProfiler) EndTransaction() {
	p.do(operation.EndTransactionID, func() {
		p.db.EndTransaction()
	})
}

func (p *ProxyProfiler) BeginBlock(number uint64) {
	p.do(operation.BeginBlockID, func() {
		p.db.BeginBlock(number)
	})
}

func (p *ProxyProfiler) EndBlock() {
	p.do(operation.EndBlockID, func() {
		p.db.EndBlock()
	})
}

func (p *ProxyProfiler) BeginSyncPeriod(number uint64) {
	p.do(operation.BeginSyncPeriodID, func() {
		p.db.BeginSyncPeriod(number)
	})
}

func (p *ProxyProfiler) EndSyncPeriod() {
	p.do(operation.EndSyncPeriodID, func() {
		p.db.EndSyncPeriod()
	})
}

// AddLog adds a log entry.
func (p *ProxyProfiler) AddLog(log *types.Log) {
	p.do(operation.AddLogID, func() {
		p.db.AddLog(log)
	})
}

// GetLogs retrieves log entries.
func (p *ProxyProfiler) GetLogs(hash common.Hash, blockHash common.Hash) (logs []*types.Log) {
	p.do(operation.GetLogsID, func() {
		logs = p.db.GetLogs(hash, blockHash)
	})
	return
}

// AddPreimage adds a SHA3 preimage.
func (p *ProxyProfiler) AddPreimage(addr common.Hash, image []byte) {
	p.do(operation.AddPreimageID, func() {
		p.db.AddPreimage(addr, image)
	})
}

// ForEachStorage performs a function over all storage locations in a contract.
func (p *ProxyProfiler) ForEachStorage(addr common.Address, fn func(common.Hash, common.Hash) bool) error {
	var err error
	p.do(operation.ForEachStorageID, func() {
		err = p.db.ForEachStorage(addr, fn)
	})
	return err
}

// Prepare sets the current transaction hash and index.
func (p *ProxyProfiler) Prepare(thash common.Hash, ti int) {
	p.do(operation.PrepareID, func() {
		p.db.Prepare(thash, ti)
	})
}

// Finalise the state in StateDB.
func (p *ProxyProfiler) Finalise(deleteEmptyObjects bool) {
	start := time.Now()
	p.db.Finalise(deleteEmptyObjects)
	elapsed := time.Since(start)
	p.ps.Profile(operation.FinaliseID, elapsed)
}

// IntermediateRoot computes the current hash of the StateDB.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (p *ProxyProfiler) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	var hash common.Hash
	p.do(operation.IntermediateRootID, func() {
		hash = p.db.IntermediateRoot(deleteEmptyObjects)
	})
	return hash
}

func (p *ProxyProfiler) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	start := time.Now()
	hash, err := p.db.Commit(deleteEmptyObjects)
	elapsed := time.Since(start)
	p.ps.Profile(operation.CommitID, elapsed)
	return hash, err
}

// GetSubstatePostAlloc gets substate post allocation.
func (p *ProxyProfiler) GetSubstatePostAlloc() substate.SubstateAlloc {
	return p.db.GetSubstatePostAlloc()
}

func (p *ProxyProfiler) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	p.db.PrepareSubstate(substate, block)
}

func (p *ProxyProfiler) Close() error {
	var err error
	p.do(operation.CloseID, func() {
		err = p.db.Close()
	})
	return err
}

func (p *ProxyProfiler) StartBulkLoad(block uint64) state.BulkLoad {
	panic("StartBulkLoad not supported by ProxyProfiler")
}

func (p *ProxyProfiler) GetArchiveState(block uint64) (state.StateDB, error) {
	return nil, fmt.Errorf("archive states are not (yet) supported by the profiling proxy")
}

func (p *ProxyProfiler) GetMemoryUsage() *state.MemoryUsage {
	return p.db.GetMemoryUsage()
}

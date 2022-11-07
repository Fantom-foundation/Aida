package tracer

import (
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/tracer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/substate"
	"math/big"
	"time"
)

// ProxyProfiler data structure for capturing and recording
// invoked StateDB operations.
type ProxyProfiler struct {
	db state.StateDB           // state db
	ps *operation.ProfileStats // operation statistics
}

// NewProxyProfiler creates a new StateDB proxy.
func NewProxyProfiler(db state.StateDB) *ProxyProfiler {
	p := new(ProxyProfiler)
	p.db = db
	p.ps = new(operation.ProfileStats)
	return p
}

// BeginBlockApply creates a new object copying state from
// the old stateDB or clears execution state of stateDB
func (p *ProxyProfiler) BeginBlockApply(root_hash common.Hash) error {
	err := p.db.BeginBlockApply(root_hash)
	return err
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
	p.db.SetNonce(addr, nonce)
}

// GetCodeHash returns the hash of the EVM bytecode.
func (p *ProxyProfiler) GetCodeHash(addr common.Address) common.Hash {
	return p.db.GetCodeHash(addr)
}

// GetCode returns the EVM bytecode of a contract.
func (p *ProxyProfiler) GetCode(addr common.Address) []byte {
	return p.db.GetCode(addr)
}

// Setcode sets the EVM bytecode of a contract.
func (p *ProxyProfiler) SetCode(addr common.Address, code []byte) {
	p.db.SetCode(addr, code)
}

// GetCodeSize returns the EVM bytecode's size.
func (p *ProxyProfiler) GetCodeSize(addr common.Address) int {
	return p.db.GetCodeSize(addr)
}

// AddRefund adds gas to the refund counter.
func (p *ProxyProfiler) AddRefund(gas uint64) {
	p.db.AddRefund(gas)
}

// SubRefund subtracts gas to the refund counter.
func (p *ProxyProfiler) SubRefund(gas uint64) {
	p.db.SubRefund(gas)
}

// GetRefund returns the current value of the refund counter.
func (p *ProxyProfiler) GetRefund() uint64 {
	return p.db.GetRefund()
}

// GetCommittedState retrieves a value that is already committed.
func (p *ProxyProfiler) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	return p.db.GetCommittedState(addr, key)
}

// GetState retrieves a value from the StateDB.
func (p *ProxyProfiler) GetState(addr common.Address, key common.Hash) common.Hash {
	return p.db.GetState(addr, key)
}

// SetState sets a value in the StateDB.
func (p *ProxyProfiler) SetState(addr common.Address, key common.Hash, value common.Hash) {
	p.db.SetState(addr, key, value)
}

// Suicide marks the given account as suicided. This clears the account balance.
// The account is still available until the state is committed;
// return a non-nil account after Suicide.
func (p *ProxyProfiler) Suicide(addr common.Address) bool {
	return p.db.Suicide(addr)
}

// HasSuicided checks whether a contract has been suicided.
func (p *ProxyProfiler) HasSuicided(addr common.Address) bool {
	return p.db.HasSuicided(addr)
}

// Exist checks whether the contract exists in the StateDB.
// Notably this also returns true for suicided accounts.
func (p *ProxyProfiler) Exist(addr common.Address) bool {
	return p.db.Exist(addr)
}

// Empty checks whether the contract is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0).
func (p *ProxyProfiler) Empty(addr common.Address) bool {
	return p.db.Empty(addr)
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
	p.db.PrepareAccessList(render, dest, precompiles, txAccesses)
}

// AddAddressToAccessList adds an address to the access list.
func (p *ProxyProfiler) AddAddressToAccessList(addr common.Address) {
	p.db.AddAddressToAccessList(addr)
}

// AddressInAccessList checks whether an address is in the access list.
func (p *ProxyProfiler) AddressInAccessList(addr common.Address) bool {
	return p.db.AddressInAccessList(addr)
}

// SlotInAccessList checks whether the (address, slot)-tuple is in the access list.
func (p *ProxyProfiler) SlotInAccessList(addr common.Address, slot common.Hash) (bool, bool) {
	addressOk, slotOk := p.db.SlotInAccessList(addr, slot)
	return addressOk, slotOk
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (p *ProxyProfiler) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	p.db.AddSlotToAccessList(addr, slot)
}

// RevertToSnapshot reverts all state changes from a given revision.
func (p *ProxyProfiler) RevertToSnapshot(snapshot int) {
	p.db.RevertToSnapshot(snapshot)
}

// Snapshot returns an identifier for the current revision of the state.
func (p *ProxyProfiler) Snapshot() int {
	snapshot := p.db.Snapshot()
	return snapshot
}

// AddLog adds a log entry.
func (p *ProxyProfiler) AddLog(log *types.Log) {
	p.db.AddLog(log)
}

// GetLogs retrieves log entries.
func (p *ProxyProfiler) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	return p.db.GetLogs(hash, blockHash)
}

// AddPreimage adds a SHA3 preimage.
func (p *ProxyProfiler) AddPreimage(addr common.Hash, image []byte) {
	p.db.AddPreimage(addr, image)
}

// ForEachStorage performs a function over all storage locations in a contract.
func (p *ProxyProfiler) ForEachStorage(addr common.Address, fn func(common.Hash, common.Hash) bool) error {
	err := p.db.ForEachStorage(addr, fn)
	return err
}

// Prepare sets the current transaction hash and index.
func (p *ProxyProfiler) Prepare(thash common.Hash, ti int) {
	p.db.Prepare(thash, ti)
}

// Finalise the state in StateDB.
func (p *ProxyProfiler) Finalise(deleteEmptyObjects bool) {
	p.db.Finalise(deleteEmptyObjects)
}

// IntermediateRoot computes the current hash of the StateDB.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (p *ProxyProfiler) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	return p.db.IntermediateRoot(deleteEmptyObjects)
}

// GetSubstatePostAlloc gets substate post allocation.
func (p *ProxyProfiler) GetSubstatePostAlloc() substate.SubstateAlloc {
	return p.db.GetSubstatePostAlloc()
}

func (p *ProxyProfiler) PrepareSubstate(substate *substate.SubstateAlloc) {
	p.db.PrepareSubstate(substate)
}

func (p *ProxyProfiler) Close() error {
	return p.db.Close()
}

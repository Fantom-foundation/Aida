package stochastic

import (
	"math/big"

	"github.com/Fantom-foundation/substate-cli/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/substate"
)

// EventProxy data structure for capturing StateDB events
type EventProxy struct {
	db       state.StateDB  // real StateDB object
	registry *EventRegistry // event registry for deriving statistical parameters
}

// NewEventProxy creates a new StateDB proxy for recording events.
func NewEventProxy(db state.StateDB, registry *EventRegistry) *EventProxy {
	return &EventProxy{db, registry}
}

// CreateAccount creates a new account.
func (p *EventProxy) CreateAccount(address common.Address) {
	// register event
	p.registry.RegisterAddressOp(StochasticCreateAccountID, &address)

	// call real StateDB
	p.db.CreateAccount(address)
}

// SubBalance subtracts amount from a contract address.
func (p *EventProxy) SubBalance(address common.Address, amount *big.Int) {
	// register event
	p.registry.RegisterAddressOp(StochasticSubBalanceID, &address)

	// call real StateDB
	p.db.SubBalance(address, amount)
}

// AddBalance adds amount to a contract address.
func (p *EventProxy) AddBalance(address common.Address, amount *big.Int) {
	// register event
	p.registry.RegisterAddressOp(StochasticAddBalanceID, &address)

	// call real StateDB
	p.db.AddBalance(address, amount)
}

// GetBalance retrieves the amount of a contract address.
func (p *EventProxy) GetBalance(address common.Address) *big.Int {
	// register event
	p.registry.RegisterAddressOp(StochasticGetBalanceID, &address)

	// call real StateDB
	return p.db.GetBalance(address)
}

// GetNonce retrieves the nonce of a contract address.
func (p *EventProxy) GetNonce(address common.Address) uint64 {
	// register event
	p.registry.RegisterAddressOp(StochasticGetNonceID, &address)

	// call real StateDB
	return p.db.GetNonce(address)
}

// SetNonce sets the nonce of a contract address.
func (p *EventProxy) SetNonce(address common.Address, nonce uint64) {
	// register event
	p.registry.RegisterAddressOp(StochasticSetNonceID, &address)

	// call real StateDB
	p.db.SetNonce(address, nonce)
}

// GetCodeHash returns the hash of the EVM bytecode.
func (p *EventProxy) GetCodeHash(address common.Address) common.Hash {
	// register event
	p.registry.RegisterAddressOp(StochasticGetCodeHashID, &address)

	// call real StateDB
	return p.db.GetCodeHash(address)
}

// GetCode returns the EVM bytecode of a contract.
func (p *EventProxy) GetCode(address common.Address) []byte {
	// register event
	p.registry.RegisterAddressOp(StochasticGetCodeID, &address)

	// call real StateDB
	return p.db.GetCode(address)
}

// Setcode sets the EVM bytecode of a contract.
func (p *EventProxy) SetCode(address common.Address, code []byte) {
	// register event
	p.registry.RegisterAddressOp(StochasticSetCodeID, &address)

	// call real StateDB
	p.db.SetCode(address, code)
}

// GetCodeSize returns the EVM bytecode's size.
func (p *EventProxy) GetCodeSize(address common.Address) int {
	// register event
	p.registry.RegisterAddressOp(StochasticGetCodeSizeID, &address)

	// call real StateDB
	return p.db.GetCodeSize(address)
}

// AddRefund adds gas to the refund counter.
func (p *EventProxy) AddRefund(gas uint64) {
	// call real StateDB
	p.db.AddRefund(gas)
}

// SubRefund subtracts gas to the refund counter.
func (p *EventProxy) SubRefund(gas uint64) {
	// call real StateDB
	p.db.SubRefund(gas)
}

// GetRefund returns the current value of the refund counter.
func (p *EventProxy) GetRefund() uint64 {
	// call real StateDB
	return p.db.GetRefund()
}

// GetCommittedState retrieves a value that is already committed.
func (p *EventProxy) GetCommittedState(address common.Address, key common.Hash) common.Hash {
	// register event
	p.registry.RegisterKeyOp(StochasticGetCommittedStateID, &address, &key)

	// call real StateDB
	return p.db.GetCommittedState(address, key)
}

// GetState retrieves a value from the StateDB.
func (p *EventProxy) GetState(address common.Address, key common.Hash) common.Hash {
	// register event
	p.registry.RegisterKeyOp(StochasticGetStateID, &address, &key)

	// call real StateDB
	return p.db.GetState(address, key)
}

// SetState sets a value in the StateDB.
func (p *EventProxy) SetState(address common.Address, key common.Hash, value common.Hash) {
	// register event
	p.registry.RegisterValueOp(StochasticSetStateID, &address, &key, &value)

	// call real StateDB
	p.db.SetState(address, key, value)
}

// Suicide an account.
func (p *EventProxy) Suicide(address common.Address) bool {
	// register event
	p.registry.RegisterAddressOp(StochasticSuicideID, &address)

	// call real StateDB
	return p.db.Suicide(address)
}

// HasSuicided checks whether a contract has been suicided.
func (p *EventProxy) HasSuicided(address common.Address) bool {
	// register event
	p.registry.RegisterAddressOp(StochasticHasSuicidedID, &address)

	// call real StateDB
	return p.db.HasSuicided(address)
}

// Exist checks whether the contract exists in the StateDB.
func (p *EventProxy) Exist(address common.Address) bool {
	// register event
	p.registry.RegisterAddressOp(StochasticExistID, &address)

	// call real StateDB
	return p.db.Exist(address)
}

// Empty checks whether the contract is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0).
func (p *EventProxy) Empty(address common.Address) bool {
	// register event
	p.registry.RegisterAddressOp(StochasticEmptyID, &address)

	// call real StateDB
	return p.db.Empty(address)
}

// PrepareAccessList handles the preparatory steps for executing a state transition.
func (p *EventProxy) PrepareAccessList(render common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	// call real StateDB
	p.db.PrepareAccessList(render, dest, precompiles, txAccesses)
}

// AddAddressToAccessList adds an address to the access list.
func (p *EventProxy) AddAddressToAccessList(address common.Address) {
	// call real StateDB
	p.db.AddAddressToAccessList(address)
}

// AddressInAccessList checks whether an address is in the access list.
func (p *EventProxy) AddressInAccessList(address common.Address) bool {
	// call real StateDB
	return p.db.AddressInAccessList(address)
}

// SlotInAccessList checks whether the (address, slot)-tuple is in the access list.
func (p *EventProxy) SlotInAccessList(address common.Address, slot common.Hash) (bool, bool) {
	// call real StateDB
	return p.db.SlotInAccessList(address, slot)
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (p *EventProxy) AddSlotToAccessList(address common.Address, slot common.Hash) {
	// call real StateDB
	p.db.AddSlotToAccessList(address, slot)
}

// RevertToSnapshot reverts all state changes from a given revision.
func (p *EventProxy) RevertToSnapshot(snapshot int) {
	// register event
	p.registry.RegisterOp(StochasticRevertToSnapshotID)

	// call real StateDB
	p.db.RevertToSnapshot(snapshot)
}

// Snapshot returns an identifier for the current revision of the state.
func (p *EventProxy) Snapshot() int {
	// register event
	p.registry.RegisterOp(StochasticSnapshotID)

	// call real StateDB
	return p.db.Snapshot()
}

// AddLog adds a log entry.
func (p *EventProxy) AddLog(log *types.Log) {
	// call real StateDB
	p.db.AddLog(log)
}

// GetLogs retrieves log entries.
func (p *EventProxy) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	// call real StateDB
	return p.db.GetLogs(hash, blockHash)
}

// AddPreimage adds a SHA3 preimage.
func (p *EventProxy) AddPreimage(address common.Hash, image []byte) {
	// call real StateDB
	p.db.AddPreimage(address, image)
}

// ForEachStorage performs a function over all storage locations in a contract.
func (p *EventProxy) ForEachStorage(address common.Address, fn func(common.Hash, common.Hash) bool) error {
	// call real StateDB
	return p.db.ForEachStorage(address, fn)
}

// Prepare sets the current transaction hash and index.
func (p *EventProxy) Prepare(thash common.Hash, ti int) {
	// call real StateDB
	p.db.Prepare(thash, ti)
}

// Finalise the state in StateDB.
func (p *EventProxy) Finalise(deleteEmptyObjects bool) {
	// register event
	p.registry.RegisterOp(StochasticFinaliseID)

	// call real StateDB
	p.db.Finalise(deleteEmptyObjects)
}

// IntermediateRoot computes the current hash of the StateDB.
func (p *EventProxy) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	// call real StateDB
	return p.db.IntermediateRoot(deleteEmptyObjects)
}

// Commit StateDB
func (p *EventProxy) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	// call real StateDB
	return p.db.Commit(deleteEmptyObjects)
}

// GetSubstatePostAlloc gets substate post allocation.
func (p *EventProxy) GetSubstatePostAlloc() substate.SubstateAlloc {
	// call real StateDB
	return p.db.GetSubstatePostAlloc()
}

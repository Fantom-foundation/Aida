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
	registry EventRegistry  // event registry for determining statistical parameters
}

// NewEventProxy creates a new StateDB proxy for recording events.
func NewEventProxy(db state.StateDB, registry EventRegistry) EventProxy {
	return EventProxy { db, registry} 
}

// CreateAccount creates a new account.
func (s *EventProxy) CreateAccount(address common.Address) {
	// register event
	s.registry.RegisterAddressOp(StochasticCreateAccountID, &address)

	// call real StateDB
	s.db.CreateAccount(address)
}

// SubBalance subtracts amount from a contract address.
func (s *EventProxy) SubBalance(address common.Address, amount *big.Int) {
	// register event
	s.registry.RegisterAddressOp(StochasticSubBalanceID, &address)

	// call real StateDB
	s.db.SubBalance(address, amount)
}

// AddBalance adds amount to a contract address.
func (s *EventProxy) AddBalance(address common.Address, amount *big.Int) {
	// register event
	s.registry.RegisterAddressOp(StochasticAddBalanceID, &address)

	// call real StateDB
	s.db.AddBalance(address, amount)
}

// GetBalance retrieves the amount of a contract address.
func (s *EventProxy) GetBalance(address common.Address) *big.Int {
	// register event
	s.registry.RegisterAddressOp(StochasticGetBalanceID, &address)

	// call real StateDB
	return s.db.GetBalance(address)
}

// GetNonce retrieves the nonce of a contract address.
func (s *EventProxy) GetNonce(address common.Address) uint64 {
	// register event
	s.registry.RegisterAddressOp(StochasticGetNonceID, &address)

	// call real StateDB
	return s.db.GetNonce(address)
}

// SetNonce sets the nonce of a contract address.
func (s *EventProxy) SetNonce(address common.Address, nonce uint64) {
	// register event
	s.registry.RegisterAddressOp(StochasticSetNonceID, &address)

	// call real StateDB
	s.db.SetNonce(address, nonce)
}

// GetCodeHash returns the hash of the EVM bytecode.
func (s *EventProxy) GetCodeHash(address common.Address) common.Hash {
	// register event
	s.registry.RegisterAddressOp(StochasticGetCodeHashID, &address)

	// call real StateDB
	return s.db.GetCodeHash(address)
}

// GetCode returns the EVM bytecode of a contract.
func (s *EventProxy) GetCode(address common.Address) []byte {
	// register event
	s.registry.RegisterAddressOp(StochasticGetCodeID, &address)

	// call real StateDB
	return s.db.GetCode(address)
}

// Setcode sets the EVM bytecode of a contract.
func (s *EventProxy) SetCode(address common.Address, code []byte) {
	// register event
	s.registry.RegisterAddressOp(StochasticSetCodeID, &address)

	// call real StateDB
	s.db.SetCode(address, code)
}

// GetCodeSize returns the EVM bytecode's size.
func (s *EventProxy) GetCodeSize(address common.Address) int {
	// register event
	s.registry.RegisterAddressOp(StochasticGetCodeSizeID, &address)

	// call real StateDB
	return s.db.GetCodeSize(address)
}

// AddRefund adds gas to the refund counter.
func (s *EventProxy) AddRefund(gas uint64) {
	// call real StateDB
	s.db.AddRefund(gas)
}

// SubRefund subtracts gas to the refund counter.
func (s *EventProxy) SubRefund(gas uint64) {
	// call real StateDB
	s.db.SubRefund(gas)
}

// GetRefund returns the current value of the refund counter.
func (s *EventProxy) GetRefund() uint64 {
	// call real StateDB
	return s.db.GetRefund()
}

// GetCommittedState retrieves a value that is already committed.
func (s *EventProxy) GetCommittedState(address common.Address, key common.Hash) common.Hash {
	// register event
	s.registry.RegisterKeyOp(StochasticGetCommittedStateID, &address, &key)

	// call real StateDB
	return s.db.GetCommittedState(address, key)
}

// GetState retrieves a value from the StateDB.
func (s *EventProxy) GetState(address common.Address, key common.Hash) common.Hash {
	// register event
	s.registry.RegisterKeyOp(StochasticGetStateID, &address, &key)

	// call real StateDB
	return s.db.GetState(address, key)
}

// SetState sets a value in the StateDB.
func (s *EventProxy) SetState(address common.Address, key common.Hash, value common.Hash) {
	// register event
	s.registry.RegisterValueOp(StochasticSetStateID, &address, &key, &value)

	// call real StateDB
	s.db.SetState(address, key, value)
}

// Suicide an account.
func (s *EventProxy) Suicide(address common.Address) bool {
	// register event
	s.registry.RegisterAddressOp(StochasticSuicideID, &address)

	// call real StateDB
	return s.db.Suicide(address)
}

// HasSuicided checks whether a contract has been suicided.
func (s *EventProxy) HasSuicided(address common.Address) bool {
	// register event
	s.registry.RegisterAddressOp(StochasticHasSuicidedID, &address)

	// call real StateDB
	return s.db.HasSuicided(address)
}

// Exist checks whether the contract exists in the StateDB.
func (s *EventProxy) Exist(address common.Address) bool {
	// register event
	s.registry.RegisterAddressOp(StochasticExistID, &address)

	// call real StateDB
	return s.db.Exist(address)
}

// Empty checks whether the contract is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0).
func (s *EventProxy) Empty(address common.Address) bool {
	// register event
	s.registry.RegisterAddressOp(StochasticEmptyID, &address)

	// call real StateDB
	return s.db.Empty(address)
}

// PrepareAccessList handles the preparatory steps for executing a state transition.
func (s *EventProxy) PrepareAccessList(render common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	// call real StateDB
	s.db.PrepareAccessList(render, dest, precompiles, txAccesses)
}

// AddAddressToAccessList adds an address to the access list.
func (s *EventProxy) AddAddressToAccessList(address common.Address) {
	// call real StateDB
	s.db.AddAddressToAccessList(address)
}

// AddressInAccessList checks whether an address is in the access list.
func (s *EventProxy) AddressInAccessList(address common.Address) bool {
	// call real StateDB
	return s.db.AddressInAccessList(address)
}

// SlotInAccessList checks whether the (address, slot)-tuple is in the access list.
func (s *EventProxy) SlotInAccessList(address common.Address, slot common.Hash) (bool, bool) {
	// call real StateDB
	return s.db.SlotInAccessList(address, slot)
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (s *EventProxy) AddSlotToAccessList(address common.Address, slot common.Hash) {
	// call real StateDB
	s.db.AddSlotToAccessList(address, slot)
}

// RevertToSnapshot reverts all state changes from a given revision.
func (s *EventProxy) RevertToSnapshot(snapshot int) {
	// register event
	s.registry.RegisterOp(StochasticRevertToSnapshotID)

	// call real StateDB
	s.db.RevertToSnapshot(snapshot)
}

// Snapshot returns an identifier for the current revision of the state.
func (s *EventProxy) Snapshot() int {
	// register event
	s.registry.RegisterOp(StochasticSnapshotID)

	// call real StateDB
	return s.db.Snapshot()
}

// AddLog adds a log entry.
func (s *EventProxy) AddLog(log *types.Log) {
	// call real StateDB
	s.db.AddLog(log)
}

// GetLogs retrieves log entries.
func (s *EventProxy) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	// call real StateDB
	return s.db.GetLogs(hash, blockHash)
}

// AddPreimage adds a SHA3 preimage.
func (s *EventProxy) AddPreimage(address common.Hash, image []byte) {
	// call real StateDB
	s.db.AddPreimage(address, image)
}

// ForEachStorage performs a function over all storage locations in a contract.
func (s *EventProxy) ForEachStorage(address common.Address, fn func(common.Hash, common.Hash) bool) error {
	// call real StateDB
	return s.db.ForEachStorage(address, fn)
}

// Prepare sets the current transaction hash and index.
func (s *EventProxy) Prepare(thash common.Hash, ti int) {
	// call real StateDB
	s.db.Prepare(thash, ti)
}

// Finalise the state in StateDB.
func (s *EventProxy) Finalise(deleteEmptyObjects bool) {
	// register event
	s.registry.RegisterOp(StochasticFinaliseID)

	// call real StateDB
	s.db.Finalise(deleteEmptyObjects)
}

// IntermediateRoot computes the current hash of the StateDB.
func (s *EventProxy) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	// call real StateDB
	return s.db.IntermediateRoot(deleteEmptyObjects)
}

// Commit StateDB
func (s *EventProxy) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	// call real StateDB
	return s.db.Commit(deleteEmptyObjects)
}

// GetSubstatePostAlloc gets substate post allocation.
func (s *EventProxy) GetSubstatePostAlloc() substate.SubstateAlloc {
	// call real StateDB
	return s.db.GetSubstatePostAlloc()
}

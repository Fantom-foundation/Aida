package stochastic

// TODO: Provide Mocking tests for proxy

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// EventProxy data structure for capturing StateDB events
type EventProxy struct {
	db        state.StateDB  // real StateDB object
	snapshots []int          // snapshot stack of currently active snapshots
	registry  *EventRegistry // event registry for deriving statistical parameters
}

// NewEventProxy creates a new StateDB proxy for recording events.
func NewEventProxy(db state.StateDB, registry *EventRegistry) *EventProxy {
	return &EventProxy{
		db:        db,
		snapshots: []int{},
		registry:  registry,
	}
}

// CreateAccount creates a new account.
func (p *EventProxy) CreateAccount(address common.Address) {
	// register event
	p.registry.RegisterAddressOp(CreateAccountID, &address)

	// call real StateDB
	p.db.CreateAccount(address)
}

// SubBalance subtracts amount from a contract address.
func (p *EventProxy) SubBalance(address common.Address, amount *big.Int) {
	// register event
	p.registry.RegisterAddressOp(SubBalanceID, &address)

	// call real StateDB
	p.db.SubBalance(address, amount)
}

// AddBalance adds amount to a contract address.
func (p *EventProxy) AddBalance(address common.Address, amount *big.Int) {
	// register event
	p.registry.RegisterAddressOp(AddBalanceID, &address)

	// call real StateDB
	p.db.AddBalance(address, amount)
}

// GetBalance retrieves the amount of a contract address.
func (p *EventProxy) GetBalance(address common.Address) *big.Int {
	// register event
	p.registry.RegisterAddressOp(GetBalanceID, &address)

	// call real StateDB
	return p.db.GetBalance(address)
}

// GetNonce retrieves the nonce of a contract address.
func (p *EventProxy) GetNonce(address common.Address) uint64 {
	// register event
	p.registry.RegisterAddressOp(GetNonceID, &address)

	// call real StateDB
	return p.db.GetNonce(address)
}

// SetNonce sets the nonce of a contract address.
func (p *EventProxy) SetNonce(address common.Address, nonce uint64) {
	// register event
	p.registry.RegisterAddressOp(SetNonceID, &address)

	// call real StateDB
	p.db.SetNonce(address, nonce)
}

// GetCodeHash returns the hash of the EVM bytecode.
func (p *EventProxy) GetCodeHash(address common.Address) common.Hash {
	// register event
	p.registry.RegisterAddressOp(GetCodeHashID, &address)

	// call real StateDB
	return p.db.GetCodeHash(address)
}

// GetCode returns the EVM bytecode of a contract.
func (p *EventProxy) GetCode(address common.Address) []byte {
	// register event
	p.registry.RegisterAddressOp(GetCodeID, &address)

	// call real StateDB
	return p.db.GetCode(address)
}

// Setcode sets the EVM bytecode of a contract.
func (p *EventProxy) SetCode(address common.Address, code []byte) {
	// register event
	p.registry.RegisterAddressOp(SetCodeID, &address)

	// call real StateDB
	p.db.SetCode(address, code)
}

// GetCodeSize returns the EVM bytecode's size.
func (p *EventProxy) GetCodeSize(address common.Address) int {
	// register event
	p.registry.RegisterAddressOp(GetCodeSizeID, &address)

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
	p.registry.RegisterKeyOp(GetCommittedStateID, &address, &key)

	// call real StateDB
	return p.db.GetCommittedState(address, key)
}

// GetState retrieves a value from the StateDB.
func (p *EventProxy) GetState(address common.Address, key common.Hash) common.Hash {
	// register event
	p.registry.RegisterKeyOp(GetStateID, &address, &key)

	// call real StateDB
	return p.db.GetState(address, key)
}

// SetState sets a value in the StateDB.
func (p *EventProxy) SetState(address common.Address, key common.Hash, value common.Hash) {
	// register event
	p.registry.RegisterValueOp(SetStateID, &address, &key, &value)

	// call real StateDB
	p.db.SetState(address, key, value)
}

// Suicide an account.
func (p *EventProxy) Suicide(address common.Address) bool {
	// register event
	p.registry.RegisterAddressOp(SuicideID, &address)

	// call real StateDB
	return p.db.Suicide(address)
}

// HasSuicided checks whether a contract has been suicided.
func (p *EventProxy) HasSuicided(address common.Address) bool {
	// register event
	p.registry.RegisterAddressOp(HasSuicidedID, &address)

	// call real StateDB
	return p.db.HasSuicided(address)
}

// Exist checks whether the contract exists in the StateDB.
func (p *EventProxy) Exist(address common.Address) bool {
	// register event
	p.registry.RegisterAddressOp(ExistID, &address)

	// call real StateDB
	return p.db.Exist(address)
}

// Empty checks whether the contract is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0).
func (p *EventProxy) Empty(address common.Address) bool {
	// register event
	p.registry.RegisterAddressOp(EmptyID, &address)

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
	p.registry.RegisterOp(RevertToSnapshotID)

	// find snapshot
	for i, recordedSnapshot := range p.snapshots {
		if recordedSnapshot == snapshot {
			// register snapshot delta
			p.registry.RegisterSnapshotDelta(len(p.snapshots) - i - 1)

			// cull all elements between found snapshot and top-of-stack
			p.snapshots = p.snapshots[0:i]
			break
		}
	}

	// call real StateDB
	p.db.RevertToSnapshot(snapshot)
}

// Snapshot returns an identifier for the current revision of the state.
func (p *EventProxy) Snapshot() int {
	// register event
	p.registry.RegisterOp(SnapshotID)

	// call real StateDB
	snapshot := p.db.Snapshot()

	// add snapshot
	p.snapshots = append(p.snapshots, snapshot)

	return snapshot
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
	p.registry.RegisterOp(FinaliseID)

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

func (p *EventProxy) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	p.db.PrepareSubstate(substate, block)
}

func (p *EventProxy) BeginTransaction(number uint32) {
	p.db.BeginTransaction(number)

	// clear all snapshots
	p.snapshots = []int{}
}

func (p *EventProxy) EndTransaction() {
	p.db.EndTransaction()

	// clear all snapshots
	p.snapshots = []int{}
}

func (p *EventProxy) BeginBlock(number uint64) {
	p.db.BeginBlock(number)
}

func (p *EventProxy) EndBlock() {
	p.db.EndBlock()
}

func (p *EventProxy) BeginEpoch(number uint64) {
	p.db.BeginEpoch(number)
}

func (p *EventProxy) EndEpoch() {
	p.db.EndEpoch()
}

// BeginBlockApply creates a new object copying state from
// the old stateDB or clears execution state of stateDB
func (p *EventProxy) BeginBlockApply() error {
	return p.db.BeginBlockApply()
}

func (p *EventProxy) Close() error {
	return p.db.Close()
}

func (p *EventProxy) StartBulkLoad() state.BulkLoad {
	panic("StartBulkLoad not supported by EventProxy")
	return nil
}

func (p *EventProxy) GetMemoryUsage() *state.MemoryUsage {
	return p.db.GetMemoryUsage()
}

func (p *EventProxy) GetArchiveState(block uint64) (state.StateDB, error) {
	return p.db.GetArchiveState(block)
}

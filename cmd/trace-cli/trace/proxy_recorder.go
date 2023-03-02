package trace

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/dictionary"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ProxyRecorder data structure for capturing and recording
// invoked StateDB operations.
type ProxyRecorder struct {
	db    state.StateDB                 // state db
	dctx  *dictionary.Context // dictionary context for decoding information
	ch    chan operation.Operation      // channel used for streaming captured operation
	debug bool
}

// NewProxyRecorder creates a new StateDB proxy.
func NewProxyRecorder(db state.StateDB, dctx *dictionary.Context, ch chan operation.Operation, debug bool) *ProxyRecorder {
	r := new(ProxyRecorder)
	r.db = db
	r.dctx = dctx
	r.ch = ch
	r.debug = debug
	return r
}

// send new operation on channel.
func (r *ProxyRecorder) send(op operation.Operation) {
	if r.debug {
		operation.Debug(r.dctx, op)
	}
	r.ch <- op
}

// CreateAccounts creates a new account.
func (r *ProxyRecorder) CreateAccount(addr common.Address) {
	cIdx := r.dctx.EncodeContract(addr)
	r.send(operation.NewCreateAccount(cIdx))
	r.db.CreateAccount(addr)
}

// SubtractBalance subtracts amount from a contract address.
func (r *ProxyRecorder) SubBalance(addr common.Address, amount *big.Int) {
	cIdx := r.dctx.EncodeContract(addr)
	r.send(operation.NewSubBalance(cIdx, amount))
	r.db.SubBalance(addr, amount)
}

// AddBalance adds amount to a contract address.
func (r *ProxyRecorder) AddBalance(addr common.Address, amount *big.Int) {
	cIdx := r.dctx.EncodeContract(addr)
	r.send(operation.NewAddBalance(cIdx, amount))
	r.db.AddBalance(addr, amount)
}

// GetBalance retrieves the amount of a contract address.
func (r *ProxyRecorder) GetBalance(addr common.Address) *big.Int {
	cIdx := r.dctx.EncodeContract(addr)
	r.send(operation.NewGetBalance(cIdx))
	balance := r.db.GetBalance(addr)
	return balance
}

// GetNonce retrieves the nonce of a contract address.
func (r *ProxyRecorder) GetNonce(addr common.Address) uint64 {
	cIdx := r.dctx.EncodeContract(addr)
	r.send(operation.NewGetNonce(cIdx))
	nonce := r.db.GetNonce(addr)
	return nonce
}

// SetNonce sets the nonce of a contract address.
func (r *ProxyRecorder) SetNonce(addr common.Address, nonce uint64) {
	cIdx := r.dctx.EncodeContract(addr)
	r.send(operation.NewSetNonce(cIdx, nonce))
	r.db.SetNonce(addr, nonce)
}

// GetCodeHash returns the hash of the EVM bytecode.
func (r *ProxyRecorder) GetCodeHash(addr common.Address) common.Hash {
	prevCIdx := r.dctx.PrevContractIndex
	cIdx := r.dctx.EncodeContract(addr)
	if prevCIdx == cIdx {
		r.send(operation.NewGetCodeHashLc())
	} else {
		r.send(operation.NewGetCodeHash(cIdx))
	}

	hash := r.db.GetCodeHash(addr)
	return hash
}

// GetCode returns the EVM bytecode of a contract.
func (r *ProxyRecorder) GetCode(addr common.Address) []byte {
	cIdx := r.dctx.EncodeContract(addr)
	r.send(operation.NewGetCode(cIdx))
	code := r.db.GetCode(addr)
	return code
}

// Setcode sets the EVM bytecode of a contract.
func (r *ProxyRecorder) SetCode(addr common.Address, code []byte) {
	cIdx := r.dctx.EncodeContract(addr)
	bcIdx := r.dctx.EncodeCode(code)
	r.send(operation.NewSetCode(cIdx, bcIdx))
	r.db.SetCode(addr, code)
}

// GetCodeSize returns the EVM bytecode's size.
func (r *ProxyRecorder) GetCodeSize(addr common.Address) int {
	cIdx := r.dctx.EncodeContract(addr)
	r.send(operation.NewGetCodeSize(cIdx))
	size := r.db.GetCodeSize(addr)
	return size
}

// AddRefund adds gas to the refund counter.
func (r *ProxyRecorder) AddRefund(gas uint64) {
	r.db.AddRefund(gas)
}

// SubRefund subtracts gas to the refund counter.
func (r *ProxyRecorder) SubRefund(gas uint64) {
	r.db.SubRefund(gas)
}

// GetRefund returns the current value of the refund counter.
func (r *ProxyRecorder) GetRefund() uint64 {
	gas := r.db.GetRefund()
	return gas
}

// GetCommittedState retrieves a value that is already committed.
func (r *ProxyRecorder) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	prevCIdx := r.dctx.PrevContractIndex
	cIdx := r.dctx.EncodeContract(addr)
	sIdx, sPos := r.dctx.EncodeStorage(key)
	if prevCIdx == cIdx && sPos == 0 {
		r.send(operation.NewGetCommittedStateLcls())
	} else {
		r.send(operation.NewGetCommittedState(cIdx, sIdx))
	}
	value := r.db.GetCommittedState(addr, key)
	return value
}

// GetState retrieves a value from the StateDB.
func (r *ProxyRecorder) GetState(addr common.Address, key common.Hash) common.Hash {
	prevCIdx := r.dctx.PrevContractIndex
	cIdx := r.dctx.EncodeContract(addr)
	sIdx, sPos := r.dctx.EncodeStorage(key)
	var op operation.Operation
	if cIdx == prevCIdx {
		if sPos == 0 {
			op = operation.NewGetStateLcls()
		} else if sPos != -1 {
			op = operation.NewGetStateLccs(sPos)
		} else {
			op = operation.NewGetStateLc(sIdx)
		}
	} else {
		op = operation.NewGetState(cIdx, sIdx)
	}
	r.send(op)
	value := r.db.GetState(addr, key)
	return value
}

// SetState sets a value in the StateDB.
func (r *ProxyRecorder) SetState(addr common.Address, key common.Hash, value common.Hash) {
	prevCIdx := r.dctx.PrevContractIndex
	cIdx := r.dctx.EncodeContract(addr)
	sIdx, sPos := r.dctx.EncodeStorage(key)
	if cIdx == prevCIdx && sPos == 0 {
		r.send(operation.NewSetStateLcls(&value))
	} else {
		r.send(operation.NewSetState(cIdx, sIdx, &value))
	}
	r.db.SetState(addr, key, value)
}

// Suicide marks the given account as suicided. This clears the account balance.
// The account is still available until the state is committed;
// return a non-nil account after Suicide.
func (r *ProxyRecorder) Suicide(addr common.Address) bool {
	cIdx := r.dctx.EncodeContract(addr)
	r.send(operation.NewSuicide(cIdx))
	ok := r.db.Suicide(addr)
	return ok
}

// HasSuicided checks whether a contract has been suicided.
func (r *ProxyRecorder) HasSuicided(addr common.Address) bool {
	hasSuicided := r.db.HasSuicided(addr)
	return hasSuicided
}

// Exist checks whether the contract exists in the StateDB.
// Notably this also returns true for suicided accounts.
func (r *ProxyRecorder) Exist(addr common.Address) bool {
	cIdx := r.dctx.EncodeContract(addr)
	r.send(operation.NewExist(cIdx))
	return r.db.Exist(addr)
}

// Empty checks whether the contract is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0).
func (r *ProxyRecorder) Empty(addr common.Address) bool {
	empty := r.db.Empty(addr)
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
func (r *ProxyRecorder) PrepareAccessList(render common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	r.db.PrepareAccessList(render, dest, precompiles, txAccesses)
}

// AddAddressToAccessList adds an address to the access list.
func (r *ProxyRecorder) AddAddressToAccessList(addr common.Address) {
	r.db.AddAddressToAccessList(addr)
}

// AddressInAccessList checks whether an address is in the access list.
func (r *ProxyRecorder) AddressInAccessList(addr common.Address) bool {
	ok := r.db.AddressInAccessList(addr)
	return ok
}

// SlotInAccessList checks whether the (address, slot)-tuple is in the access list.
func (r *ProxyRecorder) SlotInAccessList(addr common.Address, slot common.Hash) (bool, bool) {
	addressOk, slotOk := r.db.SlotInAccessList(addr, slot)
	return addressOk, slotOk
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (r *ProxyRecorder) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	r.db.AddSlotToAccessList(addr, slot)
}

// RevertToSnapshot reverts all state changes from a given revision.
func (r *ProxyRecorder) RevertToSnapshot(snapshot int) {
	r.send(operation.NewRevertToSnapshot(snapshot))
	r.db.RevertToSnapshot(snapshot)
}

// Snapshot returns an identifier for the current revision of the state.
func (r *ProxyRecorder) Snapshot() int {
	snapshot := r.db.Snapshot()
	// TODO: check overrun
	r.send(operation.NewSnapshot(int32(snapshot)))
	return snapshot
}

// AddLog adds a log entry.
func (r *ProxyRecorder) AddLog(log *types.Log) {
	r.db.AddLog(log)
}

// GetLogs retrieves log entries.
func (r *ProxyRecorder) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	return r.db.GetLogs(hash, blockHash)
}

// AddPreimage adds a SHA3 preimage.
func (r *ProxyRecorder) AddPreimage(addr common.Hash, image []byte) {
	r.db.AddPreimage(addr, image)
}

// ForEachStorage performs a function over all storage locations in a contract.
func (r *ProxyRecorder) ForEachStorage(addr common.Address, fn func(common.Hash, common.Hash) bool) error {
	err := r.db.ForEachStorage(addr, fn)
	return err
}

// Prepare sets the current transaction hash and index.
func (r *ProxyRecorder) Prepare(thash common.Hash, ti int) {
	r.db.Prepare(thash, ti)
}

// Finalise the state in StateDB.
func (r *ProxyRecorder) Finalise(deleteEmptyObjects bool) {
	r.send(operation.NewFinalise(deleteEmptyObjects))
	r.db.Finalise(deleteEmptyObjects)
}

// IntermediateRoot computes the current hash of the StateDB.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (r *ProxyRecorder) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	return r.db.IntermediateRoot(deleteEmptyObjects)
}

func (r *ProxyRecorder) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	return r.db.Commit(deleteEmptyObjects)
}

// GetSubstatePostAlloc gets substate post allocation.
func (r *ProxyRecorder) GetSubstatePostAlloc() substate.SubstateAlloc {
	return r.db.GetSubstatePostAlloc()
}

func (r *ProxyRecorder) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	r.db.PrepareSubstate(substate, block)
}

func (r *ProxyRecorder) BeginTransaction(number uint32) {
	r.db.BeginTransaction(number)
}

func (r *ProxyRecorder) EndTransaction() {
	r.db.EndTransaction()
}

func (r *ProxyRecorder) BeginBlock(number uint64) {
	r.db.BeginBlock(number)
}

func (r *ProxyRecorder) EndBlock() {
	r.db.EndBlock()
}

func (r *ProxyRecorder) BeginEpoch(number uint64) {
	r.db.BeginEpoch(number)
}

func (r *ProxyRecorder) EndEpoch() {
	r.db.EndEpoch()
}

func (r *ProxyRecorder) GetArchiveState(block uint64) (state.StateDB, error) {
	return r.db.GetArchiveState(block)
}

// BeginBlockApply creates a new object copying state from
// the old stateDB or clears execution state of stateDB
func (r *ProxyRecorder) BeginBlockApply() error {
	return r.db.BeginBlockApply()
}

func (r *ProxyRecorder) Close() error {
	return r.db.Close()
}

func (r *ProxyRecorder) StartBulkLoad() state.BulkLoad {
	panic("StartBulkLoad not supported by ProxyRecorder")
	return nil
}

func (r *ProxyRecorder) GetMemoryUsage() *state.MemoryUsage {
	return r.db.GetMemoryUsage()
}

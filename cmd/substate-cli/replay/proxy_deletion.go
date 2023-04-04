package replay

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ContractLiveness struct {
	Addr      common.Address
	IsDeleted bool //if false, the account was created
}

// ProxyDeletion data structure for capturing and recording
// invoked StateDB operations.
type ProxyDeletion struct {
	db state.StateDB // state db
	ch chan ContractLiveness
}

// NewProxyDeletion creates a new StateDB proxy.
func NewProxyDeletion(db state.StateDB, ch chan ContractLiveness) *ProxyDeletion {
	r := new(ProxyDeletion)
	r.db = db
	r.ch = ch
	return r
}

// CreateAccounts creates a new account.
func (r *ProxyDeletion) CreateAccount(addr common.Address) {
	r.db.CreateAccount(addr)
	r.ch <- ContractLiveness{Addr: addr, IsDeleted: false}
}

// SubtractBalance subtracts amount from a contract address.
func (r *ProxyDeletion) SubBalance(addr common.Address, amount *big.Int) {
	r.db.SubBalance(addr, amount)
}

// AddBalance adds amount to a contract address.
func (r *ProxyDeletion) AddBalance(addr common.Address, amount *big.Int) {
	r.db.AddBalance(addr, amount)
}

// GetBalance retrieves the amount of a contract address.
func (r *ProxyDeletion) GetBalance(addr common.Address) *big.Int {
	balance := r.db.GetBalance(addr)
	return balance
}

// GetNonce retrieves the nonce of a contract address.
func (r *ProxyDeletion) GetNonce(addr common.Address) uint64 {
	nonce := r.db.GetNonce(addr)
	return nonce
}

// SetNonce sets the nonce of a contract address.
func (r *ProxyDeletion) SetNonce(addr common.Address, nonce uint64) {
	r.db.SetNonce(addr, nonce)
}

// GetCodeHash returns the hash of the EVM bytecode.
func (r *ProxyDeletion) GetCodeHash(addr common.Address) common.Hash {
	hash := r.db.GetCodeHash(addr)
	return hash
}

// GetCode returns the EVM bytecode of a contract.
func (r *ProxyDeletion) GetCode(addr common.Address) []byte {
	code := r.db.GetCode(addr)
	return code
}

// Setcode sets the EVM bytecode of a contract.
func (r *ProxyDeletion) SetCode(addr common.Address, code []byte) {
	r.db.SetCode(addr, code)
}

// GetCodeSize returns the EVM bytecode's size.
func (r *ProxyDeletion) GetCodeSize(addr common.Address) int {
	size := r.db.GetCodeSize(addr)
	return size
}

// AddRefund adds gas to the refund counter.
func (r *ProxyDeletion) AddRefund(gas uint64) {
	r.db.AddRefund(gas)
}

// SubRefund subtracts gas to the refund counter.
func (r *ProxyDeletion) SubRefund(gas uint64) {
	r.db.SubRefund(gas)
}

// GetRefund returns the current value of the refund counter.
func (r *ProxyDeletion) GetRefund() uint64 {
	gas := r.db.GetRefund()
	return gas
}

// GetCommittedState retrieves a value that is already committed.
func (r *ProxyDeletion) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	value := r.db.GetCommittedState(addr, key)
	return value
}

// GetState retrieves a value from the StateDB.
func (r *ProxyDeletion) GetState(addr common.Address, key common.Hash) common.Hash {
	value := r.db.GetState(addr, key)
	return value
}

// SetState sets a value in the StateDB.
func (r *ProxyDeletion) SetState(addr common.Address, key common.Hash, value common.Hash) {
	r.db.SetState(addr, key, value)
}

// Suicide marks the given account as suicided. This clears the account balance.
// The account is still available until the state is committed;
// return a non-nil account after Suicide.
func (r *ProxyDeletion) Suicide(addr common.Address) bool {
	ok := r.db.Suicide(addr)
	if ok {
		r.ch <- ContractLiveness{Addr: addr, IsDeleted: true}
	}
	return ok
}

// HasSuicided checks whether a contract has been suicided.
func (r *ProxyDeletion) HasSuicided(addr common.Address) bool {
	hasSuicided := r.db.HasSuicided(addr)
	return hasSuicided
}

// Exist checks whether the contract exists in the StateDB.
// Notably this also returns true for suicided accounts.
func (r *ProxyDeletion) Exist(addr common.Address) bool {
	return r.db.Exist(addr)
}

// Empty checks whether the contract is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0).
func (r *ProxyDeletion) Empty(addr common.Address) bool {
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
func (r *ProxyDeletion) PrepareAccessList(render common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	r.db.PrepareAccessList(render, dest, precompiles, txAccesses)
}

// AddAddressToAccessList adds an address to the access list.
func (r *ProxyDeletion) AddAddressToAccessList(addr common.Address) {
	r.db.AddAddressToAccessList(addr)
}

// AddressInAccessList checks whether an address is in the access list.
func (r *ProxyDeletion) AddressInAccessList(addr common.Address) bool {
	ok := r.db.AddressInAccessList(addr)
	return ok
}

// SlotInAccessList checks whether the (address, slot)-tuple is in the access list.
func (r *ProxyDeletion) SlotInAccessList(addr common.Address, slot common.Hash) (bool, bool) {
	addressOk, slotOk := r.db.SlotInAccessList(addr, slot)
	return addressOk, slotOk
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (r *ProxyDeletion) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	r.db.AddSlotToAccessList(addr, slot)
}

// RevertToSnapshot reverts all state changes from a given revision.
func (r *ProxyDeletion) RevertToSnapshot(snapshot int) {
	r.db.RevertToSnapshot(snapshot)
}

// Snapshot returns an identifier for the current revision of the state.
func (r *ProxyDeletion) Snapshot() int {
	snapshot := r.db.Snapshot()
	return snapshot
}

// AddLog adds a log entry.
func (r *ProxyDeletion) AddLog(log *types.Log) {
	r.db.AddLog(log)
}

// GetLogs retrieves log entries.
func (r *ProxyDeletion) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	return r.db.GetLogs(hash, blockHash)
}

// AddPreimage adds a SHA3 preimage.
func (r *ProxyDeletion) AddPreimage(addr common.Hash, image []byte) {
	r.db.AddPreimage(addr, image)
}

// ForEachStorage performs a function over all storage locations in a contract.
func (r *ProxyDeletion) ForEachStorage(addr common.Address, fn func(common.Hash, common.Hash) bool) error {
	err := r.db.ForEachStorage(addr, fn)
	return err
}

// Prepare sets the current transaction hash and index.
func (r *ProxyDeletion) Prepare(thash common.Hash, ti int) {
	r.db.Prepare(thash, ti)
}

// Finalise the state in StateDB.
func (r *ProxyDeletion) Finalise(deleteEmptyObjects bool) {
	r.db.Finalise(deleteEmptyObjects)
}

// IntermediateRoot computes the current hash of the StateDB.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (r *ProxyDeletion) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	return r.db.IntermediateRoot(deleteEmptyObjects)
}

func (r *ProxyDeletion) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	return r.db.Commit(deleteEmptyObjects)
}

func (r *ProxyDeletion) Error() error {
	return r.db.Error()
}

// GetSubstatePostAlloc gets substate post allocation.
func (r *ProxyDeletion) GetSubstatePostAlloc() substate.SubstateAlloc {
	return r.db.GetSubstatePostAlloc()
}

func (r *ProxyDeletion) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	r.db.PrepareSubstate(substate, block)
}

func (r *ProxyDeletion) BeginTransaction(number uint32) {
	r.db.BeginTransaction(number)
}

func (r *ProxyDeletion) EndTransaction() {
	r.db.EndTransaction()
}

func (r *ProxyDeletion) BeginBlock(number uint64) {
	r.db.BeginBlock(number)
}

func (r *ProxyDeletion) EndBlock() {
	r.db.EndBlock()
}

func (r *ProxyDeletion) BeginEpoch(number uint64) {
	r.db.BeginEpoch(number)
}

func (r *ProxyDeletion) EndEpoch() {
	r.db.EndEpoch()
}

func (r *ProxyDeletion) GetArchiveState(block uint64) (state.StateDB, error) {
	return r.db.GetArchiveState(block)
}

func (r *ProxyDeletion) Close() error {
	return r.db.Close()
}

func (r *ProxyDeletion) StartBulkLoad() state.BulkLoad {
	panic("StartBulkLoad not supported by ProxyDeletion")
	return nil
}

func (r *ProxyDeletion) GetMemoryUsage() *state.MemoryUsage {
	return r.db.GetMemoryUsage()
}

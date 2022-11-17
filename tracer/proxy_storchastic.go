package tracer

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/substate-cli/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/substate"
)

const firstOperation = 255

// ProxyStochastic data structure for capturing and recording
// invoked StateDB operations.
type ProxyStochastic struct {
	db    state.StateDB           // state db
	dctx  *dict.DictionaryContext // dictionary context for decoding information
	debug bool
}

// NewProxyStochastic creates a new StateDB proxy.
func NewProxyStochastic(db state.StateDB, dctx *dict.DictionaryContext, debug bool) *ProxyStochastic {
	r := new(ProxyStochastic)
	r.db = db
	r.dctx = dctx
	r.debug = debug
	return r
}

// CreateAccount creates a new account.
func (r *ProxyStochastic) CreateAccount(addr common.Address) {
	r.recordOp(operation.CreateAccountID)

	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.CreateAccountID]++
	}
	_ = r.dctx.EncodeContract(addr)
	r.db.CreateAccount(addr)
}

// SubBalance subtracts amount from a contract address.
func (r *ProxyStochastic) SubBalance(addr common.Address, amount *big.Int) {
	r.recordOp(operation.SubBalanceID)

	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.SubBalanceID]++
	}

	_ = r.dctx.EncodeContract(addr)
	r.db.SubBalance(addr, amount)
}

// AddBalance adds amount to a contract address.
func (r *ProxyStochastic) AddBalance(addr common.Address, amount *big.Int) {
	r.recordOp(operation.AddBalanceID)

	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.AddBalanceID]++
	}

	_ = r.dctx.EncodeContract(addr)
	r.db.AddBalance(addr, amount)
}

// GetBalance retrieves the amount of a contract address.
func (r *ProxyStochastic) GetBalance(addr common.Address) *big.Int {
	r.recordOp(operation.GetBalanceID)

	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.GetBalanceID]++
	}

	_ = r.dctx.EncodeContract(addr)
	balance := r.db.GetBalance(addr)
	return balance
}

// GetNonce retrieves the nonce of a contract address.
func (r *ProxyStochastic) GetNonce(addr common.Address) uint64 {
	r.recordOp(operation.GetNonceID)

	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.GetNonceID]++
	}

	_ = r.dctx.EncodeContract(addr)
	nonce := r.db.GetNonce(addr)
	return nonce
}

// SetNonce sets the nonce of a contract address.
func (r *ProxyStochastic) SetNonce(addr common.Address, nonce uint64) {
	r.recordOp(operation.SetNonceID)

	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.SetNonceID]++
	}

	_ = r.dctx.EncodeContract(addr)
	r.db.SetNonce(addr, nonce)
}

// GetCodeHash returns the hash of the EVM bytecode.
func (r *ProxyStochastic) GetCodeHash(addr common.Address) common.Hash {
	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.GetCodeHashID]++
	}

	prevCIdx := r.dctx.PrevContractIndex
	cIdx := r.dctx.EncodeContract(addr)
	if prevCIdx == cIdx {
		r.recordOp(operation.GetCodeHashLcID)
	} else {
		r.recordOp(operation.GetCodeHashID)
	}

	hash := r.db.GetCodeHash(addr)
	return hash
}

// GetCode returns the EVM bytecode of a contract.
func (r *ProxyStochastic) GetCode(addr common.Address) []byte {
	r.recordOp(operation.GetCodeID)

	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.GetCodeID]++
	}

	_ = r.dctx.EncodeContract(addr)
	code := r.db.GetCode(addr)
	return code
}

// Setcode sets the EVM bytecode of a contract.
func (r *ProxyStochastic) SetCode(addr common.Address, code []byte) {
	r.recordOp(operation.SetCodeID)

	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.SetCodeID]++
	}

	_ = r.dctx.EncodeContract(addr)
	_ = r.dctx.EncodeCode(code)
	r.db.SetCode(addr, code)
}

// GetCodeSize returns the EVM bytecode's size.
func (r *ProxyStochastic) GetCodeSize(addr common.Address) int {
	r.recordOp(operation.GetCodeSizeID)

	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.GetCodeSizeID]++
	}

	_ = r.dctx.EncodeContract(addr)
	size := r.db.GetCodeSize(addr)
	return size
}

// AddRefund adds gas to the refund counter.
func (r *ProxyStochastic) AddRefund(gas uint64) {
	r.db.AddRefund(gas)
}

// SubRefund subtracts gas to the refund counter.
func (r *ProxyStochastic) SubRefund(gas uint64) {
	r.db.SubRefund(gas)
}

// GetRefund returns the current value of the refund counter.
func (r *ProxyStochastic) GetRefund() uint64 {
	gas := r.db.GetRefund()
	return gas
}

// GetCommittedState retrieves a value that is already committed.
func (r *ProxyStochastic) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.GetCommittedStateID]++
	}

	if !r.dctx.HasEncodedStorage(key) {
		r.dctx.StorageFreq[operation.GetCommittedStateID]++
	}

	prevCIdx := r.dctx.PrevContractIndex
	cIdx := r.dctx.EncodeContract(addr)
	_, sPos := r.dctx.EncodeStorage(key)
	if prevCIdx == cIdx && sPos == 0 {
		r.recordOp(operation.GetCommittedStateLclsID)
	} else {
		r.recordOp(operation.GetCommittedStateID)
	}
	value := r.db.GetCommittedState(addr, key)
	return value
}

// GetState retrieves a value from the StateDB.
func (r *ProxyStochastic) GetState(addr common.Address, key common.Hash) common.Hash {
	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.GetStateID]++
	}

	if !r.dctx.HasEncodedStorage(key) {
		r.dctx.StorageFreq[operation.GetStateID]++
	}

	prevCIdx := r.dctx.PrevContractIndex
	cIdx := r.dctx.EncodeContract(addr)
	_, sPos := r.dctx.EncodeStorage(key)
	if cIdx == prevCIdx {
		if sPos == 0 {
			r.recordOp(operation.GetStateLclsID)
		} else if sPos != -1 {
			r.recordOp(operation.GetStateLccsID)
		} else {
			r.recordOp(operation.GetStateLcID)
		}
	} else {
		r.recordOp(operation.GetStateID)
	}
	value := r.db.GetState(addr, key)
	return value
}

// SetState sets a value in the StateDB.
func (r *ProxyStochastic) SetState(addr common.Address, key common.Hash, value common.Hash) {
	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.SetStateID]++
	}

	if !r.dctx.HasEncodedStorage(key) {
		r.dctx.StorageFreq[operation.SetStateID]++
	}

	if !r.dctx.HasEncodedValue(key) {
		r.dctx.ValueFreq[operation.SetStateID]++
	}

	prevCIdx := r.dctx.PrevContractIndex
	cIdx := r.dctx.EncodeContract(addr)
	_, sPos := r.dctx.EncodeStorage(key)
	_ = r.dctx.EncodeValue(value)
	if cIdx == prevCIdx && sPos == 0 {
		r.recordOp(operation.SetStateLclsID)
	} else {
		r.recordOp(operation.SetStateID)
	}
	r.db.SetState(addr, key, value)
}

// Suicide marks the given account as suicided. This clears the account balance.
// The account is still available until the state is committed;
// return a non-nil account after Suicide.
func (r *ProxyStochastic) Suicide(addr common.Address) bool {
	r.recordOp(operation.SuicideID)

	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.SuicideID]++
	}

	_ = r.dctx.EncodeContract(addr)
	ok := r.db.Suicide(addr)
	return ok
}

// HasSuicided checks whether a contract has been suicided.
func (r *ProxyStochastic) HasSuicided(addr common.Address) bool {
	r.recordOp(operation.HasSuicidedID)

	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.HasSuicidedID]++
	}

	hasSuicided := r.db.HasSuicided(addr)
	return hasSuicided
}

// Exist checks whether the contract exists in the StateDB.
// Notably this also returns true for suicided accounts.
func (r *ProxyStochastic) Exist(addr common.Address) bool {
	r.recordOp(operation.ExistID)

	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.ExistID]++
	}

	_ = r.dctx.EncodeContract(addr)
	return r.db.Exist(addr)
}

// Empty checks whether the contract is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0).
func (r *ProxyStochastic) Empty(addr common.Address) bool {
	r.recordOp(operation.EmptyID)

	if !r.dctx.HasEncodedContract(addr) {
		r.dctx.ContractFreq[operation.EmptyID]++
	}
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
func (r *ProxyStochastic) PrepareAccessList(render common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	r.db.PrepareAccessList(render, dest, precompiles, txAccesses)
}

// AddAddressToAccessList adds an address to the access list.
func (r *ProxyStochastic) AddAddressToAccessList(addr common.Address) {
	r.db.AddAddressToAccessList(addr)
}

// AddressInAccessList checks whether an address is in the access list.
func (r *ProxyStochastic) AddressInAccessList(addr common.Address) bool {
	ok := r.db.AddressInAccessList(addr)
	return ok
}

// SlotInAccessList checks whether the (address, slot)-tuple is in the access list.
func (r *ProxyStochastic) SlotInAccessList(addr common.Address, slot common.Hash) (bool, bool) {
	addressOk, slotOk := r.db.SlotInAccessList(addr, slot)
	return addressOk, slotOk
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (r *ProxyStochastic) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	r.db.AddSlotToAccessList(addr, slot)
}

// RevertToSnapshot reverts all state changes from a given revision.
func (r *ProxyStochastic) RevertToSnapshot(snapshot int) {
	r.recordOp(operation.RevertToSnapshotID)
	r.db.RevertToSnapshot(snapshot)
}

// Snapshot returns an identifier for the current revision of the state.
func (r *ProxyStochastic) Snapshot() int {
	r.recordOp(operation.SnapshotID)
	snapshot := r.db.Snapshot()
	// TODO: check overrun
	return snapshot
}

// AddLog adds a log entry.
func (r *ProxyStochastic) AddLog(log *types.Log) {
	r.db.AddLog(log)
}

// GetLogs retrieves log entries.
func (r *ProxyStochastic) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	return r.db.GetLogs(hash, blockHash)
}

// AddPreimage adds a SHA3 preimage.
func (r *ProxyStochastic) AddPreimage(addr common.Hash, image []byte) {
	r.db.AddPreimage(addr, image)
}

// ForEachStorage performs a function over all storage locations in a contract.
func (r *ProxyStochastic) ForEachStorage(addr common.Address, fn func(common.Hash, common.Hash) bool) error {
	err := r.db.ForEachStorage(addr, fn)
	return err
}

// Prepare sets the current transaction hash and index.
func (r *ProxyStochastic) Prepare(thash common.Hash, ti int) {
	r.db.Prepare(thash, ti)
}

// Finalise the state in StateDB.
func (r *ProxyStochastic) Finalise(deleteEmptyObjects bool) {
	r.recordOp(operation.FinaliseID)
	r.db.Finalise(deleteEmptyObjects)
}

// IntermediateRoot computes the current hash of the StateDB.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (r *ProxyStochastic) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	return r.db.IntermediateRoot(deleteEmptyObjects)
}

func (r *ProxyStochastic) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	return r.db.Commit(deleteEmptyObjects)
}

// GetSubstatePostAlloc gets substate post allocation.
func (r *ProxyStochastic) GetSubstatePostAlloc() substate.SubstateAlloc {
	return r.db.GetSubstatePostAlloc()
}

func (r *ProxyStochastic) recordOp(id byte) {
	if r.dctx.PrevOpId != firstOperation {
		r.dctx.OpFreq[id]++
		r.dctx.TFreq[[2]byte{r.dctx.PrevOpId, id}]++
		r.dctx.PrevOpId = id
	} else {
		r.dctx.PrevOpId = id
	}
}

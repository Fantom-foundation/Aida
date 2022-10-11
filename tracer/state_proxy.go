package tracer

import (
	"math/big"

	"github.com/Fantom-foundation/substate-cli/state"
	"github.com/Fantom-foundation/aida/tracer/operation"
	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/substate"
)

// StateDB proxy datastructure for capturing invoked StateDB operations.
type StateProxyDB struct {
	db   state.StateDB      // state db
	dctx *dict.DictionaryContext // dictionary context for decoding information
	ch   chan operation.Operation     // channel used for streaming captured operation
}

// Create a new StateDB proxy.
func NewStateProxyDB(db state.StateDB, dctx *dict.DictionaryContext, ch chan operation.Operation) state.StateDB {
	p := new(StateProxyDB)
	p.db = db
	p.dctx = dctx
	p.ch = ch
	return p
}

// Create account an account.
func (s *StateProxyDB) CreateAccount(addr common.Address) {
	cIdx := s.dctx.EncodeContract(addr)
	s.ch <- operation.NewCreateAccount(cIdx)
	s.db.CreateAccount(addr)
}

// Subtract amount from a contract address.
func (s *StateProxyDB) SubBalance(addr common.Address, amount *big.Int) {
	s.db.SubBalance(addr, amount)
}

// Add amount to a contract address.
func (s *StateProxyDB) AddBalance(addr common.Address, amount *big.Int) {
	s.db.AddBalance(addr, amount)
}

// Obtain the amount of a contract address.
func (s *StateProxyDB) GetBalance(addr common.Address) *big.Int {
	cIdx := s.dctx.EncodeContract(addr)
	s.ch <- operation.NewGetBalance(cIdx)
	balance := s.db.GetBalance(addr)
	return balance
}

// Obtain the nonce of a contract address.
func (s *StateProxyDB) GetNonce(addr common.Address) uint64 {
	nonce := s.db.GetNonce(addr)
	return nonce
}

// Set the nonce of a contract address.
func (s *StateProxyDB) SetNonce(addr common.Address, nonce uint64) {
	s.db.SetNonce(addr, nonce)
}

// Return the hash of the EVM bytecode.
func (s *StateProxyDB) GetCodeHash(addr common.Address) common.Hash {
	cIdx := s.dctx.EncodeContract(addr)
	s.ch <- operation.NewGetCodeHash(cIdx)
	hash := s.db.GetCodeHash(addr)
	return hash
}

// Return the EVM bytecode of a contract.
func (s *StateProxyDB) GetCode(addr common.Address) []byte {
	code := s.db.GetCode(addr)
	return code
}

// Set the EVM bytecode of a contract.
func (s *StateProxyDB) SetCode(addr common.Address, code []byte) {
	s.db.SetCode(addr, code)
}

// Return the EVM bytecode's size.
func (s *StateProxyDB) GetCodeSize(addr common.Address) int {
	size := s.db.GetCodeSize(addr)
	return size
}

// Add gas to the refund counter.
func (s *StateProxyDB) AddRefund(gas uint64) {
	s.db.AddRefund(gas)
}

// Subtract gas to the refund counter.
func (s *StateProxyDB) SubRefund(gas uint64) {
	s.db.SubRefund(gas)
}

// Obtain the current value of the refund counter.
func (s *StateProxyDB) GetRefund() uint64 {
	gas := s.db.GetRefund()
	return gas
}

// Retrieve a value that is already committed.
func (s *StateProxyDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	cIdx := s.dctx.EncodeContract(addr)
	sIdx := s.dctx.EncodeStorage(key)
	s.ch <- operation.NewGetCommittedState(cIdx, sIdx)
	value := s.db.GetCommittedState(addr, key)
	return value
}

// Retrieve a value from the StateDB.
func (s *StateProxyDB) GetState(addr common.Address, key common.Hash) common.Hash {
	cIdx := s.dctx.EncodeContract(addr)
	sIdx := s.dctx.EncodeStorage(key)
	s.ch <- operation.NewGetState(cIdx, sIdx)
	value := s.db.GetState(addr, key)
	return value
}

// Set a value in the StateDB.
func (s *StateProxyDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	cIdx := s.dctx.EncodeContract(addr)
	sIdx := s.dctx.EncodeStorage(key)
	vIdx := s.dctx.EncodeValue(value)
	s.ch <- operation.NewSetState(cIdx, sIdx, vIdx)
	s.db.SetState(addr, key, value)
}

// Mark the given account as suicided. This clears the account balance.
// The account is still available until the state is committed;
// return a non-nil account after Suicide.
func (s *StateProxyDB) Suicide(addr common.Address) bool {
	cIdx := s.dctx.EncodeContract(addr)
	s.ch <- operation.NewSuicide(cIdx)
	ok := s.db.Suicide(addr)
	return ok
}

// Check whether a contract has been suicided.
func (s *StateProxyDB) HasSuicided(addr common.Address) bool {
	hasSuicided := s.db.HasSuicided(addr)
	return hasSuicided
}

// Check whether the contract exists in the StateDB.
// Notably this also returns true for suicided accounts.
func (s *StateProxyDB) Exist(addr common.Address) bool {
	cIdx := s.dctx.EncodeContract(addr)
	s.ch <- operation.NewExist(cIdx)
	return s.db.Exist(addr)
}

// Check whether the contract is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0).
func (s *StateProxyDB) Empty(addr common.Address) bool {
	empty := s.db.Empty(addr)
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
func (s *StateProxyDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.db.PrepareAccessList(sender, dest, precompiles, txAccesses)
}

// Add an address to the access list.
func (s *StateProxyDB) AddAddressToAccessList(addr common.Address) {
	s.db.AddAddressToAccessList(addr)
}

// Check whether an address is in the access list.
func (s *StateProxyDB) AddressInAccessList(addr common.Address) bool {
	ok := s.db.AddressInAccessList(addr)
	return ok
}

// Check whether the (address, slot)-tuple is in the access list.
func (s *StateProxyDB) SlotInAccessList(addr common.Address, slot common.Hash) (bool, bool) {
	addressOk, slotOk := s.db.SlotInAccessList(addr, slot)
	return addressOk, slotOk
}

// Add the given (address, slot)-tuple to the access list
func (s *StateProxyDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.db.AddSlotToAccessList(addr, slot)
}

// Revert all state changes from a given revision.
func (s *StateProxyDB) RevertToSnapshot(snapshot int) {
	s.ch <- operation.NewRevertToSnapshot(snapshot)
	s.db.RevertToSnapshot(snapshot)
}

// Return an identifier for the current revision of the state.
func (s *StateProxyDB) Snapshot() int {
	s.ch <- operation.NewSnapshot()
	snapshot := s.db.Snapshot()
	return snapshot
}

// Add a log entry.
func (s *StateProxyDB) AddLog(log *types.Log) {
	s.db.AddLog(log)
}

// Retrieve log entries.
func (s *StateProxyDB) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	return s.db.GetLogs(hash, blockHash)
}

// Adds SHA3 preimage.
func (s *StateProxyDB) AddPreimage(addr common.Hash, image []byte) {
	s.db.AddPreimage(addr, image)
}

// Performs a function over all storage locations in a contract.
func (s *StateProxyDB) ForEachStorage(addr common.Address, fn func(common.Hash, common.Hash) bool) error {
	err := s.db.ForEachStorage(addr, fn)
	return err
}

// Set the current transaction hash and index.
func (s *StateProxyDB) Prepare(thash common.Hash, ti int) {
	s.db.Prepare(thash, ti)
}

// Finalise the state in StateDB.
func (s *StateProxyDB) Finalise(deleteEmptyObjects bool) {
	s.ch <- operation.NewFinalise(deleteEmptyObjects)
	s.db.Finalise(deleteEmptyObjects)
}

// Computes the current hash of the StateDB.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (s *StateProxyDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	return s.db.IntermediateRoot(deleteEmptyObjects)
}

// Get substate post allocation.
func (s *StateProxyDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	return s.db.GetSubstatePostAlloc()
}

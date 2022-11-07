package operation

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/substate"
)

// MockStateDB data structure
type MockStateDB struct {
	msg string // signature message
}

// NewMockStateDB creates a new mock StateDB object for testing execute
func NewMockStateDB() *MockStateDB {
	return &MockStateDB{}
}

func (s *MockStateDB) BeginBlockApply(root_hash common.Hash) error {
	return nil
}

// GetSignature retrieves the call signature of the last call
func (s *MockStateDB) GetSignature() string {
	return s.msg
}

func (s *MockStateDB) CreateAccount(addr common.Address) {
	s.msg = fmt.Sprintf("CreateAccount: %v", addr.Hex())
}

func (s *MockStateDB) Exist(addr common.Address) bool {
	s.msg = fmt.Sprintf("Exist: %v", addr.Hex())
	return false
}

func (s *MockStateDB) Empty(addr common.Address) bool {
	s.msg = fmt.Sprintf("Empty: %v", addr.Hex())
	return false
}

func (s *MockStateDB) Suicide(addr common.Address) bool {
	s.msg = fmt.Sprintf("Suicide: %v", addr.Hex())
	return false
}

func (s *MockStateDB) HasSuicided(addr common.Address) bool {
	s.msg = fmt.Sprintf("HasSuicided: %v", addr.Hex())
	return false
}

func (s *MockStateDB) GetBalance(addr common.Address) *big.Int {
	s.msg = fmt.Sprintf("GetBalance: %v", addr.Hex())
	return &big.Int{}
}

func (s *MockStateDB) AddBalance(addr common.Address, value *big.Int) {
	s.msg = fmt.Sprintf("AddBalance: %v %v", addr.Hex(), value.String())
}

func (s *MockStateDB) SubBalance(addr common.Address, value *big.Int) {
	s.msg = fmt.Sprintf("SubBalance: %v %v", addr.Hex(), value.String())
}

func (s *MockStateDB) GetNonce(addr common.Address) uint64 {
	s.msg = fmt.Sprintf("GetNonce: %v", addr.Hex())
	return uint64(0)
}

func (s *MockStateDB) SetNonce(addr common.Address, value uint64) {
	s.msg = fmt.Sprintf("SetNonce: %v %v", addr.Hex(), value)
}

func (s *MockStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	s.msg = fmt.Sprintf("GetCommittedState: %v %v", addr.Hex(), key.Hex())
	return common.Hash{}
}

func (s *MockStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	s.msg = fmt.Sprintf("GetState: %v %v", addr.Hex(), key.Hex())
	return common.Hash{}
}

func (s *MockStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.msg = fmt.Sprintf("SetState: %v %v %v", addr.Hex(), key.Hex(), value.Hex())
}

func (s *MockStateDB) GetCode(addr common.Address) []byte {
	s.msg = fmt.Sprintf("GetCode: %v", addr.Hex())
	return []byte{}
}

func (s *MockStateDB) GetCodeHash(addr common.Address) common.Hash {
	s.msg = fmt.Sprintf("GetCodeHash: %v", addr.Hex())
	return common.Hash{}
}

func (s *MockStateDB) GetCodeSize(addr common.Address) int {
	s.msg = fmt.Sprintf("GetCodeSize: %v", addr.Hex())
	return 0
}

func (s *MockStateDB) SetCode(addr common.Address, code []byte) {
	s.msg = fmt.Sprintf("GetCodeSize: %v %x", addr.Hex(), code)
}

func (s *MockStateDB) Snapshot() int {
	s.msg = fmt.Sprintf("Snapshot:")
	return 0
}

func (s *MockStateDB) RevertToSnapshot(id int) {
	s.msg = fmt.Sprintf("RevertToSnapshot: %v", id)
}

func (s *MockStateDB) Finalise(deleteEmptyObjects bool) {
	s.msg = fmt.Sprintf("Finalise: %v", deleteEmptyObjects)
}

func (s *MockStateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	s.msg = fmt.Sprintf("IntermediateRoote: %v", deleteEmptyObjects)
	return common.Hash{}
}

func (s *MockStateDB) Prepare(thash common.Hash, ti int) {
	s.msg = fmt.Sprintf("Prepare: %v %v", thash, ti)
}

func (s *MockStateDB) AddRefund(gas uint64) {
	s.msg = fmt.Sprintf("AddRefund: %v", gas)
}

func (s *MockStateDB) SubRefund(gas uint64) {
	s.msg = fmt.Sprintf("SubRefund: %v", gas)
}
func (s *MockStateDB) GetRefund() uint64 {
	s.msg = fmt.Sprintf("GetRefund:")
	return uint64(0)
}
func (s *MockStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.msg = fmt.Sprintf("PrepareAccessList %v %v %v %v", sender, dest, precompiles, txAccesses)
}

func (s *MockStateDB) AddressInAccessList(addr common.Address) bool {
	s.msg = fmt.Sprintf("AddressInAccessList %v", addr)
	return false
}
func (s *MockStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	s.msg = fmt.Sprintf("SlotInAccessList %v %v", addr, slot)
	return false, false
}
func (s *MockStateDB) AddAddressToAccessList(addr common.Address) {
	s.msg = fmt.Sprintf("AddAddressToAccessList %v", addr)
}
func (s *MockStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.msg = fmt.Sprintf("AddSlotToAccessList %v %v", addr, slot)
}

func (s *MockStateDB) AddLog(log *types.Log) {
	s.msg = fmt.Sprintf("AddLog %v", log)
}
func (s *MockStateDB) AddPreimage(hash common.Hash, preimage []byte) {
	s.msg = fmt.Sprintf("AddPreimage %v", hash)
}
func (s *MockStateDB) ForEachStorage(addr common.Address, cb func(common.Hash, common.Hash) bool) error {
	s.msg = fmt.Sprintf("ForEachStorage %v", addr)
	return nil
}
func (s *MockStateDB) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	s.msg = fmt.Sprintf("GetLog %v %v", hash, blockHash)
	return nil
}

func (s *MockStateDB) PrepareSubstate(substate *substate.SubstateAlloc) {
	// ignored
}

func (s *MockStateDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	// ignored
	return substate.SubstateAlloc{}
}

func (s *MockStateDB) Close() error {
	s.msg = fmt.Sprintf("Close:")
	return nil
}

package operation

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
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

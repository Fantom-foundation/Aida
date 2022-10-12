package state

import (
	"fmt"
	"math/big"

	cc "github.com/Fantom-foundation/Carmen/go/common"
	carmen "github.com/Fantom-foundation/Carmen/go/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/substate"
)

func MakeCarmenStateDB() (StateDB, error) {
	db, err := carmen.CreateStateDB()
	if err != nil {
		return nil, err
	}
	return &carmenStateDB{db}, nil
}

type carmenStateDB struct {
	db carmen.StateDB
}

var (
	createAccountCalled bool
	existCalled         bool
	getBalanceCalled    bool
	addBalanceCalled    bool
	subBalanceCalled    bool
	getNonceCalled      bool
	setNonceCalled      bool
	getCodeHashCalled   bool
)

func (s *carmenStateDB) CreateAccount(addr common.Address) {
	if !createAccountCalled {
		fmt.Printf("Warning: CreateAccount not implemented\n")
		createAccountCalled = true
	}
}

func (s *carmenStateDB) Exist(addr common.Address) bool {
	if !existCalled {
		fmt.Printf("Warning: Exist not implemented\n")
		existCalled = true
	}
	return true
}

func (s *carmenStateDB) Empty(addr common.Address) bool {
	panic("Not implemented")
}

func (s *carmenStateDB) Suicide(addr common.Address) {
	panic("Not implemented")
}

func (s *carmenStateDB) HasSuicided(addr common.Address) bool {
	panic("Not implemented")
}

func (s *carmenStateDB) GetBalance(addr common.Address) *big.Int {
	if !getBalanceCalled {
		fmt.Printf("WARNING: GetBalance not implemented\n")
		getBalanceCalled = true
	}
	return nil
}

func (s *carmenStateDB) AddBalance(addr common.Address, value *big.Int) {
	if !addBalanceCalled {
		fmt.Printf("WARNING: AddBalance not implemented\n")
		addBalanceCalled = true
	}
}

func (s *carmenStateDB) SubBalance(addr common.Address, value *big.Int) {
	if !subBalanceCalled {
		fmt.Printf("WARNING: SubBalance not implemented\n")
		subBalanceCalled = true
	}
}

func (s *carmenStateDB) GetNonce(addr common.Address) uint64 {
	if !getNonceCalled {
		fmt.Printf("WARNING: GetNonce not implemented\n")
		getNonceCalled = true
	}
	return 0
}

func (s *carmenStateDB) SetNonce(addr common.Address, value uint64) {
	if !setNonceCalled {
		fmt.Printf("WARNING: SetNonce not implemented\n")
		setNonceCalled = true
	}
}

func (s *carmenStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(s.db.GetCommittedState(cc.Address(addr), cc.Key(key)))
}

func (s *carmenStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(s.db.GetState(cc.Address(addr), cc.Key(key)))
}

func (s *carmenStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.db.SetState(cc.Address(addr), cc.Key(key), cc.Value(value))
}

func (s *carmenStateDB) GetCodeHash(addr common.Address) common.Hash {
	if !getCodeHashCalled {
		fmt.Printf("WARNING: GetCodeHash not implemented\n")
		getCodeHashCalled = true
	}
	return common.Hash{}
}

func (s *carmenStateDB) Snapshot() int {
	return s.db.Snapshot()
}

func (s *carmenStateDB) RevertToSnapshot(id int) {
	if !getCodeHashCalled {
		fmt.Printf("WARNING: RevertToSnaphshot not implemented\n")
		getCodeHashCalled = true
	}
}

func (s *carmenStateDB) Finalise(deleteEmptyObjects bool) {
	// nothing to do
}

func (s *carmenStateDB) PrepareSubstate(substate *substate.SubstateAlloc) {
	// ignored
}

func (s *carmenStateDB) GetSubstatePostAlloc() substate.SubstateAlloc {
	return substate.SubstateAlloc{}
}

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

var getCodeHashCalled bool

func (s *carmenStateDB) CreateAccount(addr common.Address) {
	s.db.CreateAccount(cc.Address(addr))
}

func (s *carmenStateDB) Exist(addr common.Address) bool {
	return s.db.Exist(cc.Address(addr))
}

func (s *carmenStateDB) Empty(addr common.Address) bool {
	return s.db.Empty(cc.Address(addr))
}

func (s *carmenStateDB) Suicide(addr common.Address) {
	s.db.Suicide(cc.Address(addr))
}

func (s *carmenStateDB) HasSuicided(addr common.Address) bool {
	return s.db.HasSuicided(cc.Address(addr))
}

func (s *carmenStateDB) GetBalance(addr common.Address) *big.Int {
	return s.db.GetBalance(cc.Address(addr))
}

func (s *carmenStateDB) AddBalance(addr common.Address, value *big.Int) {
	s.db.AddBalance(cc.Address(addr), value)
}

func (s *carmenStateDB) SubBalance(addr common.Address, value *big.Int) {
	s.db.SubBalance(cc.Address(addr), value)
}

func (s *carmenStateDB) GetNonce(addr common.Address) uint64 {
	return s.db.GetNonce(cc.Address(addr))
}

func (s *carmenStateDB) SetNonce(addr common.Address, value uint64) {
	s.db.SetNonce(cc.Address(addr), value)
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
	s.db.RevertToSnapshot(id)
}

func (s *carmenStateDB) EndTransaction() error {
	s.db.EndTransaction()
	return nil // TODO: check for errors
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

package state

import (
	"math/big"

	geth "github.com/Fantom-foundation/substate-cli/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/substate"
)

func MakeGethInMemoryStateDB() StateDB {
	return &gethStateDb{}
}

type gethStateDb struct {
	db geth.StateDB
}

func (s *gethStateDb) CreateAccount(addr common.Address) {
	s.db.CreateAccount(addr)
}

func (s *gethStateDb) Exist(addr common.Address) bool {
	return s.db.Exist(addr)
}

func (s *gethStateDb) Empty(addr common.Address) bool {
	return s.db.Empty(addr)
}

func (s *gethStateDb) Suicide(addr common.Address) {
	s.db.Suicide(addr)
}

func (s *gethStateDb) HasSuicided(addr common.Address) bool {
	return s.db.HasSuicided(addr)
}

func (s *gethStateDb) GetBalance(addr common.Address) *big.Int {
	return s.db.GetBalance(addr)
}

func (s *gethStateDb) AddBalance(addr common.Address, value *big.Int) {
	s.db.AddBalance(addr, value)
}

func (s *gethStateDb) SubBalance(addr common.Address, value *big.Int) {
	s.db.SubBalance(addr, value)
}

func (s *gethStateDb) GetNonce(addr common.Address) uint64 {
	return s.db.GetNonce(addr)
}

func (s *gethStateDb) SetNonce(addr common.Address, value uint64) {
	s.db.SetNonce(addr, value)
}

func (s *gethStateDb) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	return s.db.GetCommittedState(addr, key)
}

func (s *gethStateDb) GetState(addr common.Address, key common.Hash) common.Hash {
	return s.db.GetState(addr, key)
}

func (s *gethStateDb) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.db.SetState(addr, key, value)
}

func (s *gethStateDb) GetCodeHash(addr common.Address) common.Hash {
	return s.db.GetCodeHash(addr)
}

func (s *gethStateDb) Snapshot() int {
	return s.db.Snapshot()
}

func (s *gethStateDb) RevertToSnapshot(id int) {
	s.db.RevertToSnapshot(id)
}

func (s *gethStateDb) Finalise(deleteEmptyObjects bool) {
	s.db.Finalise(deleteEmptyObjects)
}

func (s *gethStateDb) PrepareSubstate(substate *substate.SubstateAlloc) {
	s.db = geth.MakeInMemoryStateDB(substate)
}

func (s *gethStateDb) GetSubstatePostAlloc() substate.SubstateAlloc {
	return s.db.GetSubstatePostAlloc()
}

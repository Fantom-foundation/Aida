package state

import (
	geth "github.com/Fantom-foundation/substate-cli/state"
	"github.com/ethereum/go-ethereum/substate"
)

func MakeGethInMemoryStateDB() StateDB {
	return &gethInMemoryStateDb{}
}

type gethInMemoryStateDb struct {
	gethStateDb
}

func (s *gethInMemoryStateDb) PrepareSubstate(substate *substate.SubstateAlloc) {
	s.db = geth.MakeInMemoryStateDB(substate)
}

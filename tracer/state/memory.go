package state

import (
	"fmt"

	geth "github.com/Fantom-foundation/substate-cli/state"
	"github.com/ethereum/go-ethereum/substate"
)

func MakeGethInMemoryStateDB(variant string) (StateDB, error) {
	if variant != "" {
		return nil, fmt.Errorf("unkown variant: %v", variant)
	}
	return &gethInMemoryStateDb{}, nil
}

type gethInMemoryStateDb struct {
	gethStateDb
}

func (s *gethInMemoryStateDb) Close() error {
	// Nothing to do.
	return nil
}

func (s *gethInMemoryStateDb) PrepareSubstate(substate *substate.SubstateAlloc) {
	s.db = geth.MakeInMemoryStateDB(substate)
}

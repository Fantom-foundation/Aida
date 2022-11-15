package state

import (
	"errors"

	flat "github.com/Fantom-foundation/go-opera-fvm/gossip/evmstore/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/substate"
)

func MakeFlatStateDB(directory, variant string) (StateDB, error) {
	return nil, errors.New("is not implemented yet")
}

type flatStateDB struct {
	flat.StateDB
}

// BeginBlockApply creates a new statedb from an existing geth database
func (s *flatStateDB) BeginBlockApply(root_hash common.Hash) error {
	return errors.New("is not implemented yet")
}

// PrepareSubstate initiates the state DB for the next transaction.
func (s *flatStateDB) PrepareSubstate(*substate.SubstateAlloc) {
	return
}

// Close requests the StateDB to flush all its content to secondary storage and shut down.
// After this call no more operations will be allowed on the state.
func (s *flatStateDB) Close() error {
	return errors.New("is not implemented yet")
}

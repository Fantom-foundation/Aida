package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	statedb "github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
)

func MakeTemporaryStatePrepper() executor.Extension[*substate.Substate] {
	return temporaryStatePrepper{}
}

// temporaryStatePrepper is an extension that introduces a fresh in-memory
// StateDB instance before each transaction execution.
type temporaryStatePrepper struct {
	extension.NilExtension[*substate.Substate]
}

func (temporaryStatePrepper) PreTransaction(state executor.State[*substate.Substate], ctx *executor.Context) error {
	ctx.State = statedb.MakeInMemoryStateDB(&state.Data.InputAlloc, uint64(state.Block))
	return nil
}

package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	substate "github.com/Fantom-foundation/Substate"
)

// MakeStateDbPrepper creates an executor extension calling PrepareSubstate on
// an optional StateDB instance before each transaction of an execution. Its main
// purpose is to support Aida's in-memory DB implementation by feeding it substate
// information before each transaction in tools like `aida-vm-sdb`.
func MakeStateDbPrepper() executor.Extension[*substate.Substate] {
	return &statePrepper{}
}

type statePrepper struct {
	extension.NilExtension[*substate.Substate]
}

func (e *statePrepper) PreTransaction(state executor.State[*substate.Substate], ctx *executor.Context) error {
	if ctx != nil && ctx.State != nil && state.Data != nil {
		ctx.State.PrepareSubstate(&state.Data.InputAlloc, uint64(state.Block))
	}
	return nil
}

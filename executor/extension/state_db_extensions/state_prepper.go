package state_db_extensions

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
)

// MakeStateDbPrepper creates an executor extension calling PrepareSubstate on
// an optional StateDB instance before each transaction of an execution. Its main
// purpose is to support Aida's in-memory DB implementation by feeding it substate
// information before each transaction in tools like `aida-vm-sdb`.
func MakeStateDbPrepper() executor.Extension {
	return &statePrepper{}
}

type statePrepper struct {
	extension.NilExtension
}

func (e *statePrepper) PreTransaction(state executor.State, context *executor.Context) error {
	if context != nil && context.State != nil && state.Substate != nil {
		context.State.PrepareSubstate(&state.Substate.InputAlloc, uint64(state.Block))
	}
	return nil
}

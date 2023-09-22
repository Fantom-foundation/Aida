package state_db_extensions

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
)

// MakeStateDbPreparator creates an executor extension calling PrepareSubstate on
// an optional StateDB instance before each transaction of an execution. Its main
// purpose is to support Aida's in-memory DB implementation by feeding it substate
// information before each transaction in tools like `aida-vm-sdb`.
func MakeStateDbPreparator() executor.Extension {
	return &statePreparator{}
}

type statePreparator struct {
	extension.NilExtension
}

func (e *statePreparator) PreTransaction(state executor.State, context *executor.Context) error {
	if context != nil && context.State != nil && state.Substate != nil {
		context.State.PrepareSubstate(&state.Substate.InputAlloc, uint64(state.Block))
	}
	return nil
}

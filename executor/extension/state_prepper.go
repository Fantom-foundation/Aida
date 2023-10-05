package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
	substate "github.com/Fantom-foundation/Substate"
)

// MakeStateDbPreparator creates an executor extension calling PrepareSubstate on
// an optional StateDB instance before each transaction of an execution. Its main
// purpose is to support Aida's in-memory DB implementation by feeding it substate
// information before each transaction in tools like `aida-vm-sdb`.
func MakeStateDbPreparator() executor.Extension[*substate.Substate] {
	return &statePreparator{}
}

type statePreparator struct {
	NilExtension[*substate.Substate]
}

func (e *statePreparator) PreTransaction(state executor.State[*substate.Substate], context *executor.Context) error {
	if context != nil && context.State != nil && state.Data != nil {
		context.State.PrepareSubstate(&state.Data.InputAlloc, uint64(state.Block))
	}
	return nil
}

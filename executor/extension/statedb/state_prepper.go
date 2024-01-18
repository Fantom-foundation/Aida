package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/txcontext"
)

// MakeStateDbPrepper creates an executor extension calling PrepareSubstate on
// an optional StateDB instance before each txcontext of an execution. Its main
// purpose is to support Aida's in-memory DB implementation by feeding it substate
// information before each txcontext in tools like `aida-vm-sdb`.
func MakeStateDbPrepper() executor.Extension[txcontext.WithValidation] {
	return &statePrepper{}
}

type statePrepper struct {
	extension.NilExtension[txcontext.WithValidation]
}

func (e *statePrepper) PreTransaction(state executor.State[txcontext.WithValidation], ctx *executor.Context) error {
	if ctx != nil && ctx.State != nil && state.Data != nil {
		alloc := state.Data.GetInputState()
		ctx.State.PrepareSubstate(alloc, uint64(state.Block))
	}
	return nil
}

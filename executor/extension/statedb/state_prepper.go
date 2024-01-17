package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/executor/transaction/substate_transaction"
)

// MakeStateDbPrepper creates an executor extension calling PrepareSubstate on
// an optional StateDB instance before each transaction of an execution. Its main
// purpose is to support Aida's in-memory DB implementation by feeding it substate
// information before each transaction in tools like `aida-vm-sdb`.
func MakeStateDbPrepper() executor.Extension[substate_transaction.SubstateData] {
	return &statePrepper{}
}

type statePrepper struct {
	extension.NilExtension[substate_transaction.SubstateData]
}

func (e *statePrepper) PreTransaction(state executor.State[substate_transaction.SubstateData], ctx *executor.Context) error {
	if ctx != nil && ctx.State != nil && state.Data != nil {
		alloc := state.Data.GetInputAlloc()
		ctx.State.PrepareSubstate(alloc, uint64(state.Block))
	}
	return nil
}

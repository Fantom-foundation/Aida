package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
)

// MakeTransactionEventEmitter creates a executor.Extension to call BeginBlock() and EndBlock()
func MakeTransactionEventEmitter[T any]() executor.Extension[T] {
	return &transactionEventEmitter[T]{}
}

type transactionEventEmitter[T any] struct {
	extension.NilExtension[T]
}

func (transactionEventEmitter[T]) PreTransaction(state executor.State[T], ctx *executor.Context) error {
	return ctx.State.BeginTransaction(uint32(state.Transaction))
}

func (transactionEventEmitter[T]) PostTransaction(_ executor.State[T], ctx *executor.Context) error {
	return ctx.State.EndTransaction()
}

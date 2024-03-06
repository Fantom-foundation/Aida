package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
)

type blockEventEmitter[T any] struct {
	extension.NilExtension[T]
}

// MakeBlockEventEmitter creates a executor.Extension to call BeginBlock() and EndBlock()
func MakeBlockEventEmitter[T any]() executor.Extension[T] {
	return &blockEventEmitter[T]{}
}

func (l *blockEventEmitter[T]) PreBlock(state executor.State[T], ctx *executor.Context) error {
	return ctx.State.BeginBlock(uint64(state.Block))
}

// todo check whether begin and end tx here does not cause havoc for other impls

func (l *blockEventEmitter[T]) PreTransaction(state executor.State[T], ctx *executor.Context) error {
	return ctx.State.BeginTransaction(uint32(state.Transaction))
}

func (l *blockEventEmitter[T]) PostTransaction(_ executor.State[T], ctx *executor.Context) error {
	return ctx.State.EndTransaction()
}

func (l *blockEventEmitter[T]) PostBlock(_ executor.State[T], ctx *executor.Context) error {
	return ctx.State.EndBlock()
}

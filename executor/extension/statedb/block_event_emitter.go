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

func (l *blockEventEmitter[T]) PostBlock(_ executor.State[T], ctx *executor.Context) error {
	return ctx.State.EndBlock()
}

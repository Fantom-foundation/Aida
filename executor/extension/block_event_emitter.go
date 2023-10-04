package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
)

type blockEventEmitter[T any] struct {
	NilExtension[T]
	skipEndBlock bool // switch for vm-adb, which requires BeginBlock(), but can't call EndBlock()
}

// MakeBlockEventEmitter creates a executor.Extension to call BeginBlock() and EndBlock()
func MakeBlockEventEmitter[T any]() executor.Extension[T] {
	return &blockEventEmitter[T]{skipEndBlock: false}
}

// MakeBeginOnlyEmitter creates a executor.Extension to call beginBlock, but skips EndBlock()
func MakeBeginOnlyEmitter[T any]() executor.Extension[T] {
	return &blockEventEmitter[T]{skipEndBlock: true}
}

func (l *blockEventEmitter[T]) PreBlock(state executor.State[T], context *executor.Context) error {
	context.State.BeginBlock(uint64(state.Block))
	return nil
}

func (l *blockEventEmitter[T]) PostBlock(_ executor.State[T], context *executor.Context) error {
	if !l.skipEndBlock {
		context.State.EndBlock()
	}
	return nil
}

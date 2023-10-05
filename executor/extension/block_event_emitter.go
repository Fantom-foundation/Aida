package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
)

type blockEventEmitter struct {
	NilExtension
	skipEndBlock bool // switch for vm-adb, which requires BeginBlock(), but can't call EndBlock()
}

// MakeBlockEventEmitter creates a executor.Extension to call BeginBlock() and EndBlock()
func MakeBlockEventEmitter() executor.Extension {
	return &blockEventEmitter{skipEndBlock: false}
}

func (l *blockEventEmitter) PreBlock(state executor.State, context *executor.Context) error {
	context.State.BeginBlock(uint64(state.Block))
	return nil
}

func (l *blockEventEmitter) PostBlock(_ executor.State, context *executor.Context) error {
	if !l.skipEndBlock {
		context.State.EndBlock()
	}
	return nil
}

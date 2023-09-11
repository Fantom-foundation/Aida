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

// MakeBeginOnlyEmitter creates a executor.Extension to call beginBlock, but skips EndBlock()
func MakeBeginOnlyEmitter() executor.Extension {
	return &blockEventEmitter{skipEndBlock: true}
}

func (l *blockEventEmitter) PreBlock(state executor.State) error {
	state.State.BeginBlock(uint64(state.Block))
	return nil
}

func (l *blockEventEmitter) PostBlock(state executor.State) error {
	if !l.skipEndBlock {
		state.State.EndBlock()
	}
	return nil
}

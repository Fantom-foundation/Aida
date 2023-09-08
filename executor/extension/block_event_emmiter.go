package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
)

type blockEventEmitter struct {
	NilExtension
	skipEndBlock bool
}

// MakeBlockEventEmitter creates a executor.Extension to call beginBlock and endBlock
func MakeBlockEventEmitter() executor.Extension {
	return &blockEventEmitter{skipEndBlock: false}
}

// MakeBlockEventEmitter creates a executor.Extension to call beginBlock and endBlock
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

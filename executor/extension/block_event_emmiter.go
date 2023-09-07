package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
)

type blockEventExtension struct {
	NilExtension
	skipEndBlocks bool
}

// MakeBlockEventExtension creates a executor.Extension to call beginBlock and endBlock
func MakeBlockEventExtension(skipEndBlocks bool) executor.Extension {
	return &blockEventExtension{skipEndBlocks: skipEndBlocks}
}

func (l *blockEventExtension) PreBlock(state executor.State) error {
	state.State.BeginBlock(uint64(state.Block))
	return nil
}

func (l *blockEventExtension) PostBlock(state executor.State) error {
	if !l.skipEndBlocks {
		state.State.EndBlock()
	}
	return nil
}

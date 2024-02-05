package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
)

// txGeneratorBlockEventEmitter is an extension to call BeginBlock() and EndBlock()
// for tx generator.
type txGeneratorBlockEventEmitter[T any] struct {
	extension.NilExtension[T]
	lastBlock *uint64
}

// MakeTxGeneratorBlockEventEmitter creates a executor.Extension to call BeginBlock() and EndBlock()
// for tx generator
func MakeTxGeneratorBlockEventEmitter[T any]() executor.Extension[T] {
	return &txGeneratorBlockEventEmitter[T]{}
}

func (l *txGeneratorBlockEventEmitter[T]) PreTransaction(state executor.State[T], ctx *executor.Context) error {
	// if last block is nil, begin block for the current block
	// this is to ensure that the block is started before the first transaction
	if l.lastBlock == nil {
		ctx.State.BeginBlock(uint64(state.Block))
		blk := uint64(state.Block)
		l.lastBlock = &blk
	} else if *l.lastBlock != uint64(state.Block) {
		// if the last block is not equal to the current block, end the last block
		// and begin the current block
		ctx.State.EndBlock()
		ctx.State.BeginBlock(uint64(state.Block))
		blk := uint64(state.Block)
		l.lastBlock = &blk
	}
	return nil
}

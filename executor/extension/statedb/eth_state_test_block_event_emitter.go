package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/txcontext"
)

func MakeEthStateTestBlockEventEmitter() executor.Extension[txcontext.TxContext] {
	return ethStateBlockEventEmitter{}
}

type ethStateBlockEventEmitter struct {
	extension.NilExtension[txcontext.TxContext]
}

func (e ethStateBlockEventEmitter) PreTransaction(s executor.State[txcontext.TxContext], ctx *executor.Context) error {
	ctx.State.BeginBlock(uint64(s.Block))
	return nil
}

func (e ethStateBlockEventEmitter) PostTransaction(_ executor.State[txcontext.TxContext], ctx *executor.Context) error {
	ctx.State.EndBlock()
	return nil
}

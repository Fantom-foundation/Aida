package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/txcontext"
)

func MakeEthStateScopeTestEventEmitter() executor.Extension[txcontext.TxContext] {
	return ethStateScopeEventEmitter{}
}

type ethStateScopeEventEmitter struct {
	extension.NilExtension[txcontext.TxContext]
}

func (e ethStateScopeEventEmitter) PreTransaction(s executor.State[txcontext.TxContext], ctx *executor.Context) error {
	if err := ctx.State.BeginBlock(uint64(s.Block)); err != nil {
		return err
	}
	return ctx.State.BeginTransaction(uint32(s.Transaction))
}

func (e ethStateScopeEventEmitter) PostTransaction(_ executor.State[txcontext.TxContext], ctx *executor.Context) error {
	if err := ctx.State.EndTransaction(); err != nil {
		return err
	}
	return ctx.State.EndBlock()
}

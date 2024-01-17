package statedb

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/executor/transaction/substate_transaction"
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeProxyRecorderPrepper creates an extension which
// creates a temporary RecorderProxy before each transaction
func MakeProxyRecorderPrepper(cfg *utils.Config) executor.Extension[substate_transaction.SubstateData] {
	return makeProxyRecorderPrepper(cfg)
}

func makeProxyRecorderPrepper(cfg *utils.Config) *proxyRecorderPrepper {
	return &proxyRecorderPrepper{
		cfg: cfg,
	}
}

type proxyRecorderPrepper struct {
	extension.NilExtension[substate_transaction.SubstateData]
	cfg        *utils.Config
	rCtx       *context.Record
	syncPeriod uint64
}

func (p *proxyRecorderPrepper) PreRun(state executor.State[substate_transaction.SubstateData], _ *executor.Context) error {
	var err error
	p.rCtx, err = context.NewRecord(p.cfg.TraceFile, p.cfg.First)
	if err != nil {
		return fmt.Errorf("cannot create record context; %v", err)
	}

	p.rCtx.Debug = p.cfg.Debug

	// write the first sync period
	p.syncPeriod = uint64(state.Block) / p.cfg.SyncPeriodLength
	operation.WriteOp(p.rCtx, operation.NewBeginSyncPeriod(p.syncPeriod))

	return nil
}

func (p *proxyRecorderPrepper) PreBlock(state executor.State[substate_transaction.SubstateData], ctx *executor.Context) error {
	// calculate the syncPeriod for given block
	newSyncPeriod := uint64(state.Block) / p.cfg.SyncPeriodLength

	// loop because multiple periods could have been empty
	for p.syncPeriod < newSyncPeriod {
		operation.WriteOp(p.rCtx, operation.NewEndSyncPeriod())
		p.syncPeriod++
		operation.WriteOp(p.rCtx, operation.NewBeginSyncPeriod(p.syncPeriod))
	}

	operation.WriteOp(p.rCtx, operation.NewBeginBlock(uint64(state.Block)))
	return nil
}

// PreTransaction checks whether ctx.State has not been overwritten by temporary prepper,
// if so it creates RecorderProxy.
func (p *proxyRecorderPrepper) PreTransaction(_ executor.State[substate_transaction.SubstateData], ctx *executor.Context) error {
	// if ctx.State has not been change, no need to slow down the app by creating new Proxy
	if _, ok := ctx.State.(*proxy.RecorderProxy); ok {
		return nil
	}

	ctx.State = proxy.NewRecorderProxy(ctx.State, p.rCtx)
	return nil
}

func (p *proxyRecorderPrepper) PostBlock(executor.State[substate_transaction.SubstateData], *executor.Context) error {
	operation.WriteOp(p.rCtx, operation.NewEndBlock())
	return nil
}

func (p *proxyRecorderPrepper) PostRun(_ executor.State[substate_transaction.SubstateData], ctx *executor.Context, err error) error {
	operation.WriteOp(p.rCtx, operation.NewEndSyncPeriod())
	p.rCtx.Close()
	return nil
}

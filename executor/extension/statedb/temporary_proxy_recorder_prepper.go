package statedb

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
)

// MakeTemporaryProxyRecorderPrepper creates an extension which
// creates a temporary RecorderProxy before each transaction
func MakeTemporaryProxyRecorderPrepper(cfg *utils.Config) executor.Extension[*substate.Substate] {
	return makeTemporaryProxyRecorderPrepper(cfg)
}

func makeTemporaryProxyRecorderPrepper(cfg *utils.Config) *temporaryProxyRecorderPrepper {
	return &temporaryProxyRecorderPrepper{
		cfg: cfg,
	}
}

type temporaryProxyRecorderPrepper struct {
	extension.NilExtension[*substate.Substate]
	cfg        *utils.Config
	rCtx       *context.Record
	syncPeriod uint64
}

func (p *temporaryProxyRecorderPrepper) PreRun(state executor.State[*substate.Substate], _ *executor.Context) error {
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

func (p *temporaryProxyRecorderPrepper) PreBlock(state executor.State[*substate.Substate], _ *executor.Context) error {
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

func (p *temporaryProxyRecorderPrepper) PreTransaction(_ executor.State[*substate.Substate], ctx *executor.Context) error {
	ctx.State = proxy.NewRecorderProxy(ctx.State, p.rCtx)
	return nil
}

func (p *temporaryProxyRecorderPrepper) PostBlock(executor.State[*substate.Substate], *executor.Context) error {
	operation.WriteOp(p.rCtx, operation.NewEndBlock())
	return nil
}

func (p *temporaryProxyRecorderPrepper) PostRun(_ executor.State[*substate.Substate], ctx *executor.Context, err error) error {
	operation.WriteOp(p.rCtx, operation.NewEndSyncPeriod())
	p.rCtx.Close()
	return nil
}
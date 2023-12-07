package statedb

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeEventProxyPrepper creates an extension which records Stochastic Events.
func MakeEventProxyPrepper[T any](cfg *utils.Config) executor.Extension[T] {
	return makeEventProxyPrepper[T](cfg)
}

func makeEventProxyPrepper[T any](cfg *utils.Config) *eventProxyPrepper[T] {
	return &eventProxyPrepper[T]{
		log: logger.NewLogger(cfg.LogLevel, "Event-Prepper"),
		cfg: cfg,
	}
}

type eventProxyPrepper[T any] struct {
	extension.NilExtension[T]
	log           logger.Logger
	cfg           *utils.Config
	syncPeriod    uint64
	eventRegistry *stochastic.EventRegistry
}

func (p *eventProxyPrepper[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	p.eventRegistry = stochastic.NewEventRegistry()
	p.eventRegistry.RegisterOp(stochastic.BeginSyncPeriodID)

	if p.cfg.Output == "" {
		p.cfg.Output = "./events.json"
		p.log.Warningf("--%v is not set, setting output to %v", utils.OutputFlag.Name, p.cfg.Output)
	}

	if ctx.State != nil {
		ctx.State = stochastic.NewEventProxy(ctx.State, p.eventRegistry)
	}

	return nil
}

// PreBlock writes BeginBlock operation and End/BeginSyncPeriod if necessary
func (p *eventProxyPrepper[T]) PreBlock(state executor.State[T], _ *executor.Context) error {
	// calculate the syncPeriod for given block
	newSyncPeriod := uint64(state.Block) / p.cfg.SyncPeriodLength

	// loop because multiple periods could have been empty
	for p.syncPeriod < newSyncPeriod {
		p.eventRegistry.RegisterOp(stochastic.EndSyncPeriodID)
		p.syncPeriod++
		p.eventRegistry.RegisterOp(stochastic.BeginSyncPeriodID)
	}

	p.eventRegistry.RegisterOp(stochastic.BeginBlockID)
	return nil
}

// PreTransaction creates new EventProxy if not already.
func (p *eventProxyPrepper[T]) PreTransaction(_ executor.State[T], ctx *executor.Context) error {
	if _, ok := ctx.State.(*stochastic.EventProxy); ok {
		return nil
	}

	ctx.State = stochastic.NewEventProxy(ctx.State, p.eventRegistry)

	return nil
}

// PostBlock writes EndBlock operation.
func (p *eventProxyPrepper[T]) PostBlock(executor.State[T], *executor.Context) error {
	p.eventRegistry.RegisterOp(stochastic.EndBlockID)
	return nil
}

// PostRun writes events into a JSON file.
func (p *eventProxyPrepper[T]) PostRun(executor.State[T], *executor.Context, error) error {
	p.eventRegistry.RegisterOp(stochastic.EndSyncPeriodID)

	p.log.Noticef("Writing events into %v...", p.cfg.Output)

	f, err := os.Create(p.cfg.Output)
	if err != nil {
		return fmt.Errorf("cannot open json file; %v", err)
	}
	defer f.Close()
	j := p.eventRegistry.NewEventRegistryJSON()
	jOut, err := json.Marshal(j)
	if err != nil {
		return fmt.Errorf("cannot convert json file; %v", err)
	}

	_, err = fmt.Fprintln(f, string(jOut))
	if err != nil {
		return fmt.Errorf("cannot write into json file; %v", err)
	}

	return nil
}

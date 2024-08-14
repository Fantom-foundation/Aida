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

	if p.cfg.Output == "" {
		p.cfg.Output = "./events.json"
		p.log.Warningf("--%v is not set, setting output to %v", utils.OutputFlag.Name, p.cfg.Output)
	}

	if ctx.State != nil {
		ctx.State = stochastic.NewEventProxy(ctx.State, p.eventRegistry)
	}

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

// PostRun writes events into a JSON file.
func (p *eventProxyPrepper[T]) PostRun(executor.State[T], *executor.Context, error) error {
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

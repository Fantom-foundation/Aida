package profiler

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/tracer/profile"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeOperationProfiler creates a executor.Extension that records Operation profiling
func MakeOperationProfiler[T any](cfg *utils.Config) executor.Extension[T] {
	if !cfg.Profile {
		return extension.NilExtension[T]{}
	}
	return &operationProfiler[T]{
		cfg:           cfg,
		blockNumber:   cfg.First,
		intervalStart: cfg.First,
	}
}

type operationProfiler[T any] struct {
	extension.NilExtension[T]
	cfg           *utils.Config
	stats         *profile.Stats
	blockNumber   uint64
	intervalStart uint64
}

func (p *operationProfiler[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	ctx.State, p.stats = proxy.NewProfilerProxy(
		ctx.State,
		p.cfg.ProfileFile,
		p.cfg.LogLevel,
	)
	return nil
}

func (p *operationProfiler[T]) PostBlock(state executor.State[T], _ *executor.Context) error {
	p.blockNumber = uint64(state.Block)

	intervalEnd := p.intervalStart + p.cfg.ProfileInterval
	if p.blockNumber > intervalEnd {
		p.stats.PrintProfiling(p.intervalStart, intervalEnd)
		p.intervalStart = p.blockNumber
		p.stats.Reset()
	}

	return nil
}

func (p *operationProfiler[T]) PostRun(executor.State[T], *executor.Context, error) error {
	p.stats.PrintProfiling(p.intervalStart, p.blockNumber)
	return nil
}

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
		cfg:      cfg,
		interval: utils.NewInterval(cfg.First, cfg.Last, cfg.ProfileInterval),
	}
}

type operationProfiler[T any] struct {
	extension.NilExtension[T]
	cfg                *utils.Config
	stats              *profile.Stats
	interval           *utils.Interval
	lastProcessedBlock uint64
}

func (p *operationProfiler[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	ctx.State, p.stats = proxy.NewProfilerProxy(
		ctx.State,
		p.cfg.ProfileFile,
		p.cfg.LogLevel,
	)
	return nil
}

func (p *operationProfiler[T]) PreBlock(state executor.State[T], _ *executor.Context) error {
	if uint64(state.Block) > p.interval.End() {
		p.stats.PrintProfiling(p.interval.Start(), p.interval.End())
		p.interval.Next()
		p.stats.Reset()
	}

	return nil
}
<<<<<<< HEAD

func (p *operationProfiler[T]) PostBlock(state executor.State[T], _ *executor.Context) error {
	p.lastProcessedBlock = uint64(state.Block)
	return nil
}

func (p *operationProfiler[T]) PostRun(executor.State[T], *executor.Context, error) error {
	p.stats.PrintProfiling(p.interval.Start(), p.lastProcessedBlock)
	return nil
}
=======
>>>>>>> b216977 (remove unnecssary whitespace)

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
	return &operationProfiler[T]{cfg: cfg}
}

type operationProfiler[T any] struct {
	extension.NilExtension[T]
	cfg *utils.Config
	stats  *profile.Stats
}

func (p *operationProfiler[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	ctx.State, p.stats = proxy.NewProfilerProxy(
		ctx.State,
		p.cfg.ProfileFile,
		p.cfg.LogLevel,
	)
	return nil
}

func (p *operationProfiler[T]) PostRun(executor.State[T], *executor.Context, error) error {
	p.stats.PrintProfiling(p.cfg.First, p.cfg.Last)
	return nil
}

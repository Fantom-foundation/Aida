package profiler

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/tracer/profile"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeOperationProfiler creates a executor.Extension that records Operation profiling
func MakeOperationProfiler[T any](config *utils.Config) executor.Extension[T] {
	if !config.Profile {
		return extension.NilExtension[T]{}
	}
	return &operationProfiler[T]{config: config}
}

type operationProfiler[T any] struct {
	extension.NilExtension[T]
	config *utils.Config
	stats  *profile.Stats
}

func (p *operationProfiler[T]) PreRun(_ executor.State[T], context *executor.Context) error {
	context.State, p.stats = proxy.NewProfilerProxy(
		context.State,
		p.config.ProfileFile,
		p.config.LogLevel,
	)
	return nil
}

func (p *operationProfiler[T]) PostRun(executor.State[T], *executor.Context, error) error {
	p.stats.PrintProfiling(p.config.First, p.config.Last)
	return nil
}

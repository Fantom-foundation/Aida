package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeCpuProfiler creates a executor.Extension that records CPU profiling
// data for the duration between the begin and end of the execution run, if
// enabled in the provided configuration.
func MakeCpuProfiler[T any](config *utils.Config) executor.Extension[T] {
	if config.CPUProfile == "" {
		return NilExtension[T]{}
	}
	return &cpuProfiler[T]{config: config}
}

type cpuProfiler[T any] struct {
	NilExtension[T]
	config *utils.Config
}

func (p *cpuProfiler[T]) PreRun(executor.State[T], *executor.Context) error {
	return utils.StartCPUProfile(p.config)
}

func (p *cpuProfiler[T]) PostRun(executor.State[T], *executor.Context, error) error {
	utils.StopCPUProfile(p.config)
	return nil
}

package profiler

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeMemoryProfiler creates an executor.Extension that records memory profiling data if enabled in the configuration.
func MakeMemoryProfiler[T any](config *utils.Config) executor.Extension[T] {
	if config.MemoryProfile == "" {
		return extension.NilExtension[T]{}
	}
	return &memoryProfiler[T]{config: config}
}

type memoryProfiler[T any] struct {
	extension.NilExtension[T]
	config *utils.Config
}

func (p *memoryProfiler[T]) PostRun(executor.State[T], *executor.Context, error) error {
	return utils.StartMemoryProfile(p.config)
}

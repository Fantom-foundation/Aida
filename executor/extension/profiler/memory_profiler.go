package profiler

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeMemoryProfiler creates an executor.Extension that records memory profiling data if enabled in the configuration.
func MakeMemoryProfiler[T any](cfg *utils.Config) executor.Extension[T] {
	if cfg.MemoryProfile == "" {
		return extension.NilExtension[T]{}
	}
	return &memoryProfiler[T]{cfg: cfg}
}

type memoryProfiler[T any] struct {
	extension.NilExtension[T]
	cfg *utils.Config
}

func (p *memoryProfiler[T]) PostRun(executor.State[T], *executor.Context, error) error {
	return utils.StartMemoryProfile(p.cfg)
}

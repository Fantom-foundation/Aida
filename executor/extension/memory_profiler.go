package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeMemoryProfiler creates an executor.Extension that records memory profiling data if enabled in the configuration.
func MakeMemoryProfiler(config *utils.Config) executor.Extension {
	if config.MemoryProfile == "" {
		return NilExtension{}
	}
	return &memoryProfiler{config: config}
}

type memoryProfiler struct {
	NilExtension
	config *utils.Config
}

func (p *memoryProfiler) PostRun(executor.State, *executor.Context, error) error {
	return utils.StartMemoryProfile(p.config)
}

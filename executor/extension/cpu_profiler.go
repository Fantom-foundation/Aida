package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeCpuProfiler creates a executor.Extension that records CPU profiling
// data for the duration between the begin and end of the execution run, if
// enabled in the provided configuration.
func MakeCpuProfiler(config *utils.Config) executor.Extension {
	if config.CPUProfile == "" {
		return NilExtension{}
	}
	return &cpuProfiler{config: config}
}

type cpuProfiler struct {
	NilExtension
	config *utils.Config
}

func (p *cpuProfiler) PreRun(_ executor.State) error {
	return utils.StartCPUProfile(p.config)
}

func (p *cpuProfiler) PostRun(_ executor.State, _ error) error {
	utils.StopCPUProfile(p.config)
	return nil
}

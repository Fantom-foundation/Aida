package extension

import (
	"fmt"
	"os"
	"runtime/pprof"

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
	sequenceNumber int
}

func (p *cpuProfiler[T]) PreRun(executor.State[T], *executor.Context) error {
	filename := p.config.CPUProfile
	if p.config.CPUProfilePerInterval {
		p.sequenceNumber = state.Block / 100_000
		filename = p.getFileNameFor(p.sequenceNumber)
	}
	return startCpuProfiler(filename)
}

func (p *cpuProfiler[T]) PreBlock(state executor.State, _ *executor.Context) error {
	if !p.config.CPUProfilePerInterval {
		return nil
	}
	number := state.Block / 100_000
	if p.sequenceNumber == number {
		return nil
	}
	stopCpuProfiler()
	p.sequenceNumber = number
	return startCpuProfiler(p.getFileNameFor(number))
}

func (p *cpuProfiler[T]) PostRun(executor.State[T], *executor.Context, error) error {
	stopCpuProfiler()
	return nil
}

func (p *cpuProfiler[T]) getFileNameFor(sequenceNumber int) string {
	return fmt.Sprintf("%s_%05d", p.config.CPUProfile, sequenceNumber)
}

func startCpuProfiler(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create CPU profile: %s", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		return fmt.Errorf("could not start CPU profile: %s", err)
	}
	return nil
}

func stopCpuProfiler() {
	pprof.StopCPUProfile()
}

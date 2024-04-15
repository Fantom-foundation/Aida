// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package profiler

import (
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeCpuProfiler creates a executor.Extension that records CPU profiling
// data for the duration between the begin and end of the execution run, if
// enabled in the provided configuration.
func MakeCpuProfiler[T any](cfg *utils.Config) executor.Extension[T] {
	if cfg.CPUProfile == "" {
		return extension.NilExtension[T]{}
	}
	return &cpuProfiler[T]{cfg: cfg}
}

type cpuProfiler[T any] struct {
	extension.NilExtension[T]
	cfg            *utils.Config
	sequenceNumber int
}

func (p *cpuProfiler[T]) PreRun(state executor.State[T], _ *executor.Context) error {
	filename := p.cfg.CPUProfile
	if p.cfg.CPUProfilePerInterval {
		p.sequenceNumber = state.Block / 100_000
		filename = p.getFileNameFor(p.sequenceNumber)
	}
	return startCpuProfiler(filename)
}

func (p *cpuProfiler[T]) PreBlock(state executor.State[T], _ *executor.Context) error {
	if !p.cfg.CPUProfilePerInterval {
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
	return fmt.Sprintf("%s_%05d", p.cfg.CPUProfile, sequenceNumber)
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

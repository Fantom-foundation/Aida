// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func StartCPUProfile(cfg *Config) error {
	if cfg.CPUProfile != "" {
		f, err := os.Create(cfg.CPUProfile)
		if err != nil {
			return fmt.Errorf("could not create CPU profile: %s", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("could not start CPU profile: %s", err)
		}
	}
	return nil
}

func StopCPUProfile(cfg *Config) {
	if cfg.CPUProfile != "" {
		pprof.StopCPUProfile()
	}
}

func StartMemoryProfile(cfg *Config) error {
	// write memory profile if requested
	if cfg.MemoryProfile != "" {
		f, err := os.Create(cfg.MemoryProfile)
		if err != nil {
			return fmt.Errorf("could not create memory profile: %s", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			return fmt.Errorf("could not write memory profile: %s", err)
		}
	}
	return nil
}

// MemoryBreakdown prints memory usage details of statedb if applicable
func MemoryBreakdown(db state.StateDB, cfg *Config, log logger.Logger) {
	if cfg.MemoryBreakdown {
		if usage := db.GetMemoryUsage(); usage.Breakdown != nil {
			log.Noticef("State DB memory usage: %d byte\n%s", usage.UsedBytes, usage.Breakdown)
		} else {
			log.Noticef("Memory usage summary is unavailable. The selected storage solution: %v variant: %v, may not support memory breakdowns.", cfg.DbImpl, cfg.DbVariant)
		}
	}
}

// PrintEvmStatistics prints EVM implementation specific statical information
// to the console. Does nothing, if such information is not offered.
func PrintEvmStatistics(cfg *Config) {
	pvm, ok := tosca.GetInterpreter(cfg.VmImpl).(tosca.ProfilingInterpreter)
	if pvm != nil && ok {
		pvm.DumpProfile()
	}
}

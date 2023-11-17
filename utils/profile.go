package utils

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Tosca/go/vm"
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
			log.Notice("Memory usage summary is unavailable. The selected storage solution, %v variant: %v, may not support memory breakdowns.", cfg.DbImpl, cfg.DbVariant)
		}
	}
}

// PrintEvmStatistics prints EVM implementation specific stastical information
// to the console. Does nothing, if such information is not offered.
func PrintEvmStatistics(cfg *Config) {
	pvm, ok := vm.GetVirtualMachine(cfg.VmImpl).(vm.ProfilingVM)
	if pvm != nil && ok {
		pvm.DumpProfile()
	}
}

package blockprocessor

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/tracer/profile"
	"github.com/Fantom-foundation/Aida/utils"
)

// ProfileExtension provide the logging action for block processing
type ProfileExtension struct {
	ProcessorExtensions

	// state for profiling StateDB operations
	dbStats          *profile.Stats
	lastDbStatsBlock uint64
}

// NewProfileExtension creates a new logging action for block processing.
func NewProfileExtension() *ProfileExtension {
	return &ProfileExtension{}
}

// Init opens the CPU profiler if specied in the cli.
func (ext *ProfileExtension) Init(bp *BlockProcessor) error {
	// CPU profiling (if enabled)
	if err := utils.StartCPUProfile(bp.Cfg); err != nil {
		return fmt.Errorf("failed to open CPU profiler; %v", err)
	}
	return nil
}

// PostPrepare initialises state and reports on disk usage after priming.
func (ext *ProfileExtension) PostPrepare(bp *BlockProcessor) error {
	// print memory usage after priming/preparing
	utils.MemoryBreakdown(bp.Db, bp.Cfg, bp.Log)

	// is StateDb profiling switched on
	if bp.Cfg.Profile {
		bp.Db, ext.dbStats = proxy.NewProfilerProxy(bp.Db, bp.Cfg.ProfileFile, bp.Cfg.LogLevel)
	}

	return nil
}

// PostBlock issues periodic stateDB reports.
func (ext *ProfileExtension) PostBlock(bp *BlockProcessor) error {
	// initialise the last-block variables for the first time to suppress block report
	// at the beginning (in case the user has specified a large enough starting block)
	if ext.lastDbStatsBlock == 0 {
		ext.lastDbStatsBlock = bp.Block - (bp.Block % bp.Cfg.ProfileInterval)
	}

	// issue periodic StateDB report
	if bp.Cfg.Profile {
		if bp.Block-ext.lastDbStatsBlock >= bp.Cfg.ProfileInterval {
			// print dbStats
			if err := ext.dbStats.PrintProfiling(ext.lastDbStatsBlock, bp.Block); err != nil {
				return err
			}

			// reset stats in proxy
			ext.dbStats.Reset()
			ext.lastDbStatsBlock = bp.Block
		}
	}

	return nil
}

func (ext *ProfileExtension) PostTransaction(bp *BlockProcessor) error {
	return nil
}

// PostProcessing issues a summary report.
func (ext *ProfileExtension) PostProcessing(bp *BlockProcessor) error {

	// write memory profile
	if err := utils.StartMemoryProfile(bp.Cfg); err != nil {
		return err
	}

	// final block profile report
	if bp.Cfg.Profile && bp.Block != ext.lastDbStatsBlock {
		if err := ext.dbStats.PrintProfiling(ext.lastDbStatsBlock, bp.Block); err != nil {
			return err
		}
	}

	return nil
}

// Exit stops CPU profiling and issues disk report
func (ext *ProfileExtension) Exit(bp *BlockProcessor) error {
	utils.StopCPUProfile(bp.Cfg)
	return nil
}

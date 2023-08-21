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
	if err := utils.StartCPUProfile(bp.cfg); err != nil {
		return fmt.Errorf("failed to open CPU profiler; %v", err)
	}
	return nil
}

// PostPrepare initialises state and reports on disk usage after priming.
func (ext *ProfileExtension) PostPrepare(bp *BlockProcessor) error {
	// print memory usage after priming/preparing
	utils.MemoryBreakdown(bp.db, bp.cfg, bp.log)

	// is StateDb profiling switched on
	if bp.cfg.Profile {
		bp.db, ext.dbStats = proxy.NewProfilerProxy(bp.db, bp.cfg.ProfileFile, bp.cfg.LogLevel)
	}

	return nil
}

func (ext *ProfileExtension) PostBlock(bp *BlockProcessor) error {
	return nil
}

// PostTransaction issues periodic stateDB reports.
func (ext *ProfileExtension) PostTransaction(bp *BlockProcessor) error {

	// initialise the last-block variables for the first time to suppress block report
	// at the beginning (in case the user has specified a large enough starting block)
	if ext.lastDbStatsBlock == 0 {
		ext.lastDbStatsBlock = bp.block - (bp.block % bp.cfg.ProfileInterval)
	}

	// issue periodic StateDB report
	if bp.cfg.Profile {
		if bp.block-ext.lastDbStatsBlock >= bp.cfg.ProfileInterval {
			// print dbStats
			if err := ext.dbStats.PrintProfiling(ext.lastDbStatsBlock, bp.block); err != nil {
				return err
			}

			// reset stats in proxy
			ext.dbStats.Reset()
			ext.lastDbStatsBlock = bp.block
		}
	}
	return nil
}

// PostProcessing issues a summary report.
func (ext *ProfileExtension) PostProcessing(bp *BlockProcessor) error {

	// write memory profile
	if err := utils.StartMemoryProfile(bp.cfg); err != nil {
		return err
	}

	// final block profile report
	if bp.cfg.Profile && bp.block != ext.lastDbStatsBlock {
		if err := ext.dbStats.PrintProfiling(ext.lastDbStatsBlock, bp.block); err != nil {
			return err
		}
	}

	return nil
}

// Exit stops CPU profiling and issues disk report
func (ext *ProfileExtension) Exit(bp *BlockProcessor) error {
	utils.StopCPUProfile(bp.cfg)
	return nil
}

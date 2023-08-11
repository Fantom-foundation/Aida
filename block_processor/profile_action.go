package blockprocessor

import (
	"fmt"

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
func (pa *ProfileExtension) Init(bp *BlockProcessor) error {
	// CPU profiling (if enabled)
	if err := utils.StartCPUProfile(bp.cfg); err != nil {
		return fmt.Errorf("failed to open CPU profiler; %v", err)
	}
	return nil
}

// PostPrepare initialises state and reports on disk usage after priming.
func (pa *ProfileExtension) PostPrepare(bp *BlockProcessor) error {
	// print memory usage after priming/preparing
	utils.MemoryBreakdown(bp.db, bp.cfg, bp.log)

	// is StateDb profiling switched on
	if bp.cfg.Profile {
		bp.db, pa.dbStats = NewProxyProfiler(bp.db, bp.cfg.ProfileFile)
	}

	return nil
}

// PostTransactions issues periodic stateDB reports.
func (pa *ProfileExtension) PostTransaction(bp *BlockProcessor) error {

	// initialise the last-block variables for the first time to suppress block report
	// at the beginning (in case the user has specified a large enough starting block)
	if pa.lastDbStatsBlock == 0 {
		pa.lastDbStatsBlock = bp.block
	}

	// issue periodic StateDB report
	if bp.cfg.Profile {
		if bp.block-pa.lastDbStatsBlock >= bp.cfg.ProfileInterval {
			// print dbStats
			if err := pa.dbStats.PrintProfiling(pa.lastDbStatsBlock, bp.block); err != nil {
				return err
			}
			// reset stats in proxy
			pa.dbStats.Reset()
			pa.lastDbStatsBlock = bp.block
		}
	}
	return nil
}

// PostProcessing issues a summary report.
func (pa *ProfileExtension) PostProcessing(bp *BlockProcessor) error {

	// write memory profile
	if err := utils.StartMemoryProfile(bp.cfg); err != nil {
		return err
	}

	// final block profile report
	if bp.cfg.Profile && bp.block != pa.lastDbStatsBlock {
		if err := pa.dbStats.PrintProfiling(pa.lastDbStatsBlock, bp.block); err != nil {
			return err
		}
	}

	return nil
}

// Exit stops CPU profiling and issues disk report
func (la *ProfileExtension) Exit(bp *BlockProcessor) error {
	utils.StopCPUProfile(bp.cfg)
	return nil
}

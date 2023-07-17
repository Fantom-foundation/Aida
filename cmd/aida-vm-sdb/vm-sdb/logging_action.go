package runvm

import (
	"math/big"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/tracer/profile"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/google/martian/log"
)

// Time constants for reports
const (
	TimePeriod         = 15 * time.Second // when to issue a periodic report
	BlockPeriod uint64 = 100_000          // when to issue a block report
)

// LoggingAction provide the logging action for block processing
type LoggingAction struct {
	ProcessorActions

	processingStart time.Time // start time

	// state for periodic logging
	lastTimeReport       time.Time // last time when a log line was printed
	lastTimeProcessedTx  uint64    // number of transactions processed
	lastTimeProcessedGas *big.Int  // gas processed

	// state for block range logging
	lastBlockReport       time.Time // last time when a block
	lastBlockProcessedTx  uint64    // number of transactions processed
	lastBlockProcessedGas *big.Int  // gas processed
	lastBlock             uint64    // block number of last block report

	// state for profiling StateDB operations
	dbStats          *profile.Stats
	lastDbStatsBlock uint64
}

// NewLoggingAction creates a new logging action for block processing.
func NewLoggingAction() *LoggingAction {
	return &LoggingAction{
		lastTimeProcessedGas:  new(big.Int),
		lastBlockProcessedGas: new(big.Int),
	}
}

// Init opens the CPU profiler if specied in the cli.
func (la *LoggingAction) Init(bp *BlockProcessor) error {
	// CPU profiling (if enabled)
	if err := utils.StartCPUProfile(bp.cfg); err != nil {
		bp.log.Notice("Failed to open CPU profiler; %v", err)
		return err
	}
	return nil
}

// Initialise state and report on disk usage after priming.
func (la *LoggingAction) PostPrepare(bp *BlockProcessor) error {
	// print memory usage after priming/preparing
	utils.MemoryBreakdown(bp.db, bp.cfg, bp.log)

	// is StateDb profiling switched on
	if bp.cfg.Profile {
		bp.db, la.dbStats = NewProxyProfiler(bp.db, bp.cfg.ProfileFile)
	}

	// time time for block and periodic report
	la.lastTimeReport = time.Now()
	la.lastBlockReport = time.Now()
	la.processingStart = time.Now()

	return nil
}

// PostTransactions issues periodic, block, and stateDB reports.
func (la *LoggingAction) PostTransaction(bp *BlockProcessor) error {

	// initialise the last-block variables for the first time to suppress block report
	// at the beginning (in case the user has specified a large enough starting block)
	if la.lastBlock == 0 {
		la.lastBlock = bp.block
	}
	if la.lastDbStatsBlock == 0 {
		la.lastDbStatsBlock = bp.block
	}

	// suppress reports when quiet flag is enabled
	if !bp.cfg.Quiet {

		// issue a report after a time interval of length "TimePeriod"
		elapsed := time.Since(la.lastTimeReport)
		if elapsed >= TimePeriod {
			gasUsed, _ := new(big.Float).SetInt(new(big.Int).Sub(bp.totalGas, la.lastTimeProcessedGas)).Float64()
			txRate := float64(bp.totalTx-la.lastTimeProcessedTx) / (float64(elapsed.Nanoseconds()) / 1e9)
			gasRate := gasUsed / (float64(elapsed.Nanoseconds()) / 1e9)
			hours, minutes, seconds := logger.ParseTime(elapsed)
			bp.log.Infof("Elapsed time: %vh %vm %vs, at block %v (~ %.0f Tx/s, ~ %.0f Gas/s)", hours, minutes, seconds, bp.block, txRate, gasRate)
			la.lastTimeReport = time.Now()
			la.lastTimeProcessedTx = bp.totalTx
			la.lastTimeProcessedGas.Set(bp.totalGas)
		}

		// issue a report after a block range of length "BlockPeriod"
		if bp.block-la.lastBlock >= BlockPeriod {
			elapsed := time.Since(la.lastBlockReport)
			gasUsed, _ := new(big.Float).SetInt(new(big.Int).Sub(bp.totalGas, la.lastBlockProcessedGas)).Float64()
			txRate := float64(bp.totalTx-la.lastBlockProcessedTx) / (float64(elapsed.Nanoseconds()) / 1e9)
			gasRate := gasUsed / (float64(elapsed.Nanoseconds()) / 1e9)
			memoryUsage := bp.db.GetMemoryUsage()
			diskUsage := utils.GetDirectorySize(bp.stateDbDir)
			bp.log.Noticef("Reached block %d using ~ %d bytes of memory, ~ %d bytes of disk, last interval rate ~ %.0f Tx/s, ~ %.0f Gas/s",
				bp.block, memoryUsage.UsedBytes, diskUsage, txRate, gasRate)
			la.lastBlock = bp.block
			la.lastBlockReport = time.Now()
			la.lastBlockProcessedTx = bp.totalTx
			la.lastBlockProcessedGas.Set(bp.totalGas)
		}
	}

	// issue periodic StateDB report
	if bp.cfg.Profile {
		if bp.block-la.lastDbStatsBlock >= bp.cfg.ProfileInterval {
			// print dbStats
			if err := la.dbStats.PrintProfiling(la.lastDbStatsBlock, bp.block); err != nil {
				return err
			}
			// reset stats in proxy
			la.dbStats.Reset()
			la.lastDbStatsBlock = bp.block
		}
	}
	return nil
}

// PostProcessing issues a summary report.
func (la *LoggingAction) PostProcessing(bp *BlockProcessor) error {

	// write memory profile
	if err := utils.StartMemoryProfile(bp.cfg); err != nil {
		return err
	}

	// final block profile report
	if bp.cfg.Profile && bp.block != la.lastDbStatsBlock {
		if err := la.dbStats.PrintProfiling(la.lastDbStatsBlock, bp.block); err != nil {
			return err
		}
	}

	// print progress summary
	if !bp.cfg.Quiet {
		utils.MemoryBreakdown(bp.db, bp.cfg, bp.log)

		elapsed := time.Since(la.processingStart)
		gasUsed, _ := new(big.Float).SetInt(new(big.Int).Sub(bp.totalGas, la.lastBlockProcessedGas)).Float64()
		txRate := float64(bp.totalTx) / (float64(elapsed.Nanoseconds()) / 1e9)
		hours, minutes, seconds := logger.ParseTime(time.Since(la.processingStart))
		blocks := bp.cfg.Last - bp.cfg.First + 1
		log.Infof("Total elapsed time: %vh %vm %vs, processed %v blocks, %v transactions (~ %.1f Tx/s) (~ %.1f Gas/s)\n",
			hours, minutes, seconds, blocks, bp.totalTx, txRate, gasUsed)
	}

	return nil
}

// Exit stops CPU profiling and issues disk report
func (la *LoggingAction) Exit(bp *BlockProcessor) error {
	if !bp.cfg.Quiet {
		log.Infof("Final disk usage: %v MiB\n", float32(utils.GetDirectorySize(bp.stateDbDir))/float32(1024*1024))
	}
	utils.StopCPUProfile(bp.cfg)
	return nil
}

package blockprocessor

import (
	"math/big"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
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

}

// NewLoggingAction creates a new logging action for block processing.
func NewLoggingAction() *LoggingAction {
	return &LoggingAction{
		lastTimeProcessedGas:  new(big.Int),
		lastBlockProcessedGas: new(big.Int),
	}
}

// Init prepares Logging Action.
func (la *LoggingAction) Init(bp *BlockProcessor) error {
	return nil
}

// PostPrepare starts timers.
func (la *LoggingAction) PostPrepare(bp *BlockProcessor) error {
	// time time for block and periodic report
	la.lastTimeReport = time.Now()
	la.lastBlockReport = time.Now()
	la.processingStart = time.Now()

	return nil
}

// PostTransactions issues periodic, block, and stateDB memory reports.
func (la *LoggingAction) PostTransaction(bp *BlockProcessor) error {
	// suppress reports when quiet flag is enabled
	if bp.cfg.Quiet {
		return nil
	}

	// initialise the last-block variables for the first time to suppress block report
	// at the beginning (in case the user has specified a large enough starting block)
	if la.lastBlock == 0 {
		la.lastBlock = bp.block
	}

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
	return nil
}

// PostProcessing issues a summary report.
func (la *LoggingAction) PostProcessing(bp *BlockProcessor) error {
	// suppress reports when quiet flag is enabled
	if bp.cfg.Quiet {
		return nil
	}

	// print progress summary
	elapsed := time.Since(la.processingStart)
	gasUsed, _ := new(big.Float).SetInt(new(big.Int).Sub(bp.totalGas, la.lastBlockProcessedGas)).Float64()
	txRate := float64(bp.totalTx) / (float64(elapsed.Nanoseconds()) / 1e9)
	hours, minutes, seconds := logger.ParseTime(time.Since(la.processingStart))
	blocks := bp.cfg.Last - bp.cfg.First + 1
	bp.log.Infof("Total elapsed time: %vh %vm %vs, processed %v blocks, %v transactions (~ %.1f Tx/s) (~ %.1f Gas/s)\n",
		hours, minutes, seconds, blocks, bp.totalTx, txRate, gasUsed)

	return nil
}

// Exit issues disk report
func (la *LoggingAction) Exit(bp *BlockProcessor) error {
	if !bp.cfg.Quiet {
		bp.log.Infof("Final disk usage: %v MiB\n", float32(utils.GetDirectorySize(bp.stateDbDir))/float32(1024*1024))
	}
	return nil
}

package blockprocessor

import (
	"math/big"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

// Time constants for reports
const (
	BlockPeriod uint64 = 100_000 // when to issue a block report
)

// ProgressReportExtension provide the logging action for block processing
type ProgressReportExtension struct {
	ProcessorExtensions

	processingStart time.Time // start time

	// state for block range logging
	lastBlockReport       time.Time // last time when a block
	lastBlockProcessedTx  uint64    // number of transactions processed
	lastBlockProcessedGas *big.Int  // gas processed
	lastBlock             uint64    // block number of last block report

}

// NewProgressReportExtension creates a new logging action for block processing.
func NewProgressReportExtension() *ProgressReportExtension {
	return &ProgressReportExtension{
		lastBlockProcessedGas: new(big.Int),
	}
}

// Init prepares ProgressReport Extension.
func (la *ProgressReportExtension) Init(bp *BlockProcessor) error {
	la.lastBlock = bp.block - (bp.block % BlockPeriod)
	return nil
}

// PostPrepare starts timers.
func (la *ProgressReportExtension) PostPrepare(bp *BlockProcessor) error {
	// time time for block and periodic report
	la.lastBlockReport = time.Now()
	la.processingStart = time.Now()

	return nil
}

// PostTransactions issues periodic, block, and stateDB memory reports.
func (la *ProgressReportExtension) PostTransaction(bp *BlockProcessor) error {
	// suppress reports when quiet flag is enabled
	if bp.cfg.Quiet {
		return nil
	}

	// initialise the last-block variables for the first time to suppress block report
	// at the beginning (in case the user has specified a large enough starting block)
	if la.lastBlock == 0 {
		la.lastBlock = bp.block - (bp.block % BlockPeriod)
	}

	// issue a report after a block range of length "BlockPeriod"
	if bp.block-la.lastBlock >= BlockPeriod {
		elapsed := time.Since(la.lastBlockReport)
		gasUsed, _ := new(big.Float).SetInt(new(big.Int).Sub(bp.totalGas, la.lastBlockProcessedGas)).Float64()
		txRate := float64(bp.totalTx-la.lastBlockProcessedTx) / (float64(elapsed.Nanoseconds()) / 1e9)
		gasRate := gasUsed / (float64(elapsed.Nanoseconds()) / 1e9) / 1e6 // convert to MGas
		memoryUsage := float64(bp.db.GetMemoryUsage().UsedBytes) / 1024 / 1024 / 1024
		diskUsage := float64(utils.GetDirectorySize(bp.stateDbDir)) / 1024 / 1024 / 1024
		hours, minutes, seconds := logger.ParseTime(time.Since(la.processingStart))
		bp.log.Infof("Elapsed time: %d:%02d:%02d; reached block %d using ~ %0.2f GiB of memory, ~ %0.2f GiB of disk, last interval rate ~ %.2f Tx/s, ~ %.2f MGas/s",
			hours, minutes, seconds, bp.block, memoryUsage, diskUsage, txRate, gasRate)
		la.lastBlock = bp.block
		la.lastBlockReport = time.Now()
		la.lastBlockProcessedTx = bp.totalTx
		la.lastBlockProcessedGas.Set(bp.totalGas)
	}
	return nil
}

// PostProcessing issues a summary report.
func (la *ProgressReportExtension) PostProcessing(bp *BlockProcessor) error {
	// suppress reports when quiet flag is enabled
	if bp.cfg.Quiet {
		return nil
	}

	// print progress summary
	elapsed := time.Since(la.processingStart)
	gasUsed, _ := new(big.Float).SetInt(bp.totalGas).Float64()
	gasRate := gasUsed / (float64(elapsed.Nanoseconds()) / 1e9) / 1e6 // convert to MGas
	txRate := float64(bp.totalTx) / (float64(elapsed.Nanoseconds()) / 1e9)
	hours, minutes, seconds := logger.ParseTime(time.Since(la.processingStart))
	blocks := bp.cfg.Last - bp.cfg.First + 1
	bp.log.Infof("Total elapsed time: %d:%02d:%02d, processed %v blocks, %v transactions (~ %.2f Tx/s) (~ %.2f MGas/s)\n",
		hours, minutes, seconds, blocks, bp.totalTx, txRate, gasRate)

	return nil
}

// Exit issues disk report
func (la *ProgressReportExtension) Exit(bp *BlockProcessor) error {
	if !bp.cfg.Quiet {
		bp.log.Infof("Final disk usage: %v GiB\n", float32(utils.GetDirectorySize(bp.stateDbDir))/float32(1024*1024*1024))
	}
	return nil
}

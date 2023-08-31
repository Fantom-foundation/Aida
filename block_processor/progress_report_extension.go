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
	gasUsed               *big.Float
	lastBlockInitialized  bool      // true if the last block got initialized, false otherwise

}

// NewProgressReportExtension creates a new logging action for block processing.
func NewProgressReportExtension() *ProgressReportExtension {
	return &ProgressReportExtension{
		lastBlockProcessedGas: new(big.Int),
		gasUsed:               new(big.Float),
	}
}

// Init prepares ProgressReport Extension.
func (ext *ProgressReportExtension) Init(bp *BlockProcessor) error {
	if bp.Cfg.Quiet {
		return nil
	}
	ext.lastBlock = bp.Block - (bp.Block % BlockPeriod)
	return nil
}

// PostPrepare starts timers.
func (ext *ProgressReportExtension) PostPrepare(bp *BlockProcessor) error {
	if bp.Cfg.Quiet {
		return nil
	}
	// time for block and periodic report
	ext.lastBlockReport = time.Now()
	ext.processingStart = time.Now()

	return nil
}

func (ext *ProgressReportExtension) PostBlock(bp *BlockProcessor) error {
	// ignored.
	return nil
}

func (ext *ProgressReportExtension) PostTransaction(bp *BlockProcessor) error {
	return nil
}

// PostBlock issues periodic, block, and stateDB memory reports.
func (ext *ProgressReportExtension) PostBlock(bp *BlockProcessor) error {
	if bp.Cfg.Quiet {
		return nil
	}
	block := bp.Block
	totalTx := bp.TotalTx.Uint64()

	// initialise the last-block variables for the first time to suppress block report
	// at the beginning (in case the user has specified a large enough starting block)
	boundary := block - (block % BlockPeriod)
	if !ext.lastBlockInitialized {
		ext.lastBlock = boundary
		ext.lastBlockInitialized = true
	}

	// issue a report after a block range of length "BlockPeriod"
	if block-ext.lastBlock >= BlockPeriod {
		elapsed := time.Since(ext.lastBlockReport)
		gasUsed, _ := ext.gasUsed.SetInt(new(big.Int).Sub(bp.TotalGas, ext.lastBlockProcessedGas)).Float64()
		txRate := float64(totalTx-ext.lastBlockProcessedTx) / (float64(elapsed.Nanoseconds()) / 1e9)
		gasRate := gasUsed / (float64(elapsed.Nanoseconds()) / 1e9) / 1e6 // convert to MGas
		memoryUsage := float64(bp.Db.GetMemoryUsage().UsedBytes) / 1024 / 1024 / 1024
		diskUsage := float64(utils.GetDirectorySize(bp.stateDbDir)) / 1024 / 1024 / 1024
		hours, minutes, seconds := logger.ParseTime(time.Since(ext.processingStart))

		// Note: when modifying the output format here make sure to update the
		// parser of this line in scripts/run_throughput_eval.rb as well.
		bp.Log.Infof("Elapsed time: %d:%02d:%02d; reached block %d using ~ %0.2f GiB of memory, ~ %0.2f GiB of disk, last interval rate ~ %.2f Tx/s, ~ %.2f MGas/s",
			hours, minutes, seconds, block, memoryUsage, diskUsage, txRate, gasRate)

		ext.lastBlock = boundary
		ext.lastBlockReport = time.Now()
		ext.lastBlockProcessedTx = totalTx
		ext.lastBlockProcessedGas.Set(bp.TotalGas)
	}

	return nil
}

// PostProcessing issues a summary report.
func (ext *ProgressReportExtension) PostProcessing(bp *BlockProcessor) error {
	if bp.Cfg.Quiet {
		return nil
	}
	totalTx := bp.TotalTx.Uint64()

	// print progress summary
	elapsed := time.Since(ext.processingStart)
	gasUsed, _ := ext.gasUsed.SetInt(bp.TotalGas).Float64()
	gasRate := gasUsed / (float64(elapsed.Nanoseconds()) / 1e9) / 1e6 // convert to MGas
	txRate := float64(totalTx) / (float64(elapsed.Nanoseconds()) / 1e9)
	hours, minutes, seconds := logger.ParseTime(time.Since(ext.processingStart))
	blocks := bp.Cfg.Last - bp.Cfg.First + 1

	bp.Log.Infof("Total elapsed time: %d:%02d:%02d, processed %v blocks, %v transactions (~ %.2f Tx/s) (~ %.2f MGas/s)\n",
		hours, minutes, seconds, blocks, bp.TotalTx, txRate, gasRate)

	return nil
}

// Exit issues disk report
func (ext *ProgressReportExtension) Exit(bp *BlockProcessor) error {
	if bp.Cfg.Quiet {
		return nil
	}
	bp.Log.Infof("Final disk usage: %v GiB\n", float32(utils.GetDirectorySize(bp.stateDbDir))/float32(1024*1024*1024))
	return nil
}

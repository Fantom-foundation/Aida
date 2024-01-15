package tracker

import (
	"fmt"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/executor/transaction"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

const (
	ProgressTrackerDefaultReportFrequency = 100_000 // in blocks
	progressTrackerReportFormat           = "Track: block %d, memory %d, disk %d, interval_blk_rate %.2f, interval_tx_rate %.2f, interval_gas_rate %.2f, overall_blk_rate %.2f, overall_tx_rate %.2f, overall_gas_rate %.2f"
)

// MakeProgressTracker creates a progressTracker that depends on the
// PostBlock event and is only useful as part of a sequential evaluation.
func MakeProgressTracker(cfg *utils.Config, reportFrequency int) executor.Extension[transaction.SubstateData] {
	if !cfg.TrackProgress {
		return extension.NilExtension[transaction.SubstateData]{}
	}

	if reportFrequency == 0 {
		reportFrequency = ProgressTrackerDefaultReportFrequency
	}

	return makeProgressTracker(cfg, reportFrequency, logger.NewLogger(cfg.LogLevel, "ProgressTracker"))
}

func makeProgressTracker(cfg *utils.Config, reportFrequency int, log logger.Logger) *progressTracker {
	return &progressTracker{
		cfg:               cfg,
		log:               log,
		reportFrequency:   reportFrequency,
		lastReportedBlock: int(cfg.First) - (int(cfg.First) % reportFrequency),
	}
}

// progressTracker logs progress every XXX blocks depending on reportFrequency.
// Default is 100_000 blocks. This is mainly used for gathering information about process.
type progressTracker struct {
	extension.NilExtension[transaction.SubstateData]
	cfg                 *utils.Config
	log                 logger.Logger
	reportFrequency     int
	lastReportedBlock   int
	startOfRun          time.Time
	startOfLastInterval time.Time
	overallInfo         processInfo
	lastIntervalInfo    processInfo
	lock                sync.Mutex
}

type processInfo struct {
	numTransactions uint64
	gas             uint64
}

func (t *progressTracker) PreRun(state executor.State[transaction.SubstateData], _ *executor.Context) error {
	now := time.Now()
	t.startOfRun = now
	t.startOfLastInterval = now
	return nil
}

// PostTransaction increments number of transactions and saves gas used in last substate.
func (t *progressTracker) PostTransaction(state executor.State[transaction.SubstateData], _ *executor.Context) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.overallInfo.numTransactions++
	t.overallInfo.gas += state.Data.GetResult().GetGasUsed()

	return nil
}

// PostBlock registers the completed block and may trigger the logging of an update.
func (t *progressTracker) PostBlock(state executor.State[transaction.SubstateData], ctx *executor.Context) error {
	boundary := state.Block - (state.Block % t.reportFrequency)

	if state.Block-t.lastReportedBlock < t.reportFrequency {
		return nil
	}

	now := time.Now()
	overall := now.Sub(t.startOfRun)
	interval := now.Sub(t.startOfLastInterval)

	// quickly get a snapshot of the current overall progress
	t.lock.Lock()
	info := t.overallInfo
	t.lock.Unlock()

	disk, err := utils.GetDirectorySize(ctx.StateDbPath)
	if err != nil {
		return fmt.Errorf("cannot size of state-db (%v); %v", ctx.StateDbPath, err)
	}
	m := ctx.State.GetMemoryUsage()

	memory := uint64(0)
	if m != nil {
		memory = m.UsedBytes
	}

	intervalBlkRate := float64(t.reportFrequency) / interval.Seconds()
	intervalTxRate := float64(info.numTransactions-t.lastIntervalInfo.numTransactions) / interval.Seconds()
	intervalGasRate := float64(info.gas-t.lastIntervalInfo.gas) / interval.Seconds()
	t.lastIntervalInfo = info

	overallBlkRate := float64(state.Block-int(t.cfg.First)) / overall.Seconds()
	overallTxRate := float64(info.numTransactions) / overall.Seconds()
	overallGasRate := float64(info.gas) / overall.Seconds()

	t.log.Noticef(
		progressTrackerReportFormat,
		boundary, memory, disk,
		intervalBlkRate, intervalTxRate, intervalGasRate,
		overallBlkRate, overallTxRate, overallGasRate,
	)

	t.lastReportedBlock = boundary
	t.startOfLastInterval = now

	return nil
}

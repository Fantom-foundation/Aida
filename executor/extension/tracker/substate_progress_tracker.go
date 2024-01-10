package tracker

import (
	"fmt"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
)

const substateProgressTrackerReportFormat = "Track: block %d, memory %d, disk %d, interval_blk_rate %.2f, interval_tx_rate %.2f, interval_gas_rate %.2f, overall_blk_rate %.2f, overall_tx_rate %.2f, overall_gas_rate %.2f"

// MakeSubstateProgressTracker creates a substateProgressTracker that depends on the
// PostBlock event and is only useful as part of a sequential evaluation.
func MakeSubstateProgressTracker(cfg *utils.Config, reportFrequency int) executor.Extension[*substate.Substate] {
	if !cfg.TrackProgress {
		return extension.NilExtension[*substate.Substate]{}
	}

	if reportFrequency == 0 {
		reportFrequency = ProgressTrackerDefaultReportFrequency
	}

	return makeSubstateProgressTracker(cfg, reportFrequency, logger.NewLogger(cfg.LogLevel, "ProgressTracker"))
}

func makeSubstateProgressTracker(cfg *utils.Config, reportFrequency int, log logger.Logger) *substateProgressTracker {
	return &substateProgressTracker{
		progressTracker:   newProgressTracker[*substate.Substate](cfg, reportFrequency, log),
		lastReportedBlock: int(cfg.First) - (int(cfg.First) % reportFrequency),
	}
}

// substateProgressTracker logs progress every XXX blocks depending on reportFrequency.
// Default is 100_000 blocks. This is mainly used for gathering information about process.
type substateProgressTracker struct {
	*progressTracker[*substate.Substate]
	overallInfo       substateProcessInfo
	lastIntervalInfo  substateProcessInfo
	lastReportedBlock int
}

type substateProcessInfo struct {
	numTransactions uint64
	gas             uint64
}

func (t *substateProgressTracker) PreRun(executor.State[*substate.Substate], *executor.Context) error {
	now := time.Now()
	t.startOfRun = now
	t.startOfLastInterval = now
	return nil
}

// PostTransaction increments number of transactions and saves gas used in last substate.
func (t *substateProgressTracker) PostTransaction(state executor.State[*substate.Substate], _ *executor.Context) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.overallInfo.numTransactions++
	t.overallInfo.gas += state.Data.Result.GasUsed

	return nil
}

// PostBlock registers the completed block and may trigger the logging of an update.
func (t *substateProgressTracker) PostBlock(state executor.State[*substate.Substate], ctx *executor.Context) error {
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
		substateProgressTrackerReportFormat,
		boundary, memory, disk,
		intervalBlkRate, intervalTxRate, intervalGasRate,
		overallBlkRate, overallTxRate, overallGasRate,
	)

	t.lastReportedBlock = boundary
	t.startOfLastInterval = now

	return nil
}

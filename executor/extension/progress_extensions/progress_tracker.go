package progress_extensions

import (
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

const (
	ProgressTrackerDefaultReportFrequency = 100_000 // in blocks
	progressTrackerReportFormat           = "Track: block %d, memory %d, disk %d, interval_tx_rate %.2f, interval_gas_rate %.2f, overall_tx_rate %.2f, overall_gas_rate %.2f"
)

// MakeProgressTracker creates a progressTracker that depends on the
// PostBlock event and is only useful as part of a sequential evaluation.
func MakeProgressTracker(config *utils.Config, reportFrequency int) executor.Extension {
	if !config.TrackProgress {
		return extension.NilExtension{}
	}

	if reportFrequency == 0 {
		reportFrequency = ProgressTrackerDefaultReportFrequency
	}

	return makeProgressTracker(config, reportFrequency, logger.NewLogger(config.LogLevel, "ProgressTracker"))
}

func makeProgressTracker(config *utils.Config, reportFrequency int, log logger.Logger) *progressTracker {
	return &progressTracker{
		config:            config,
		log:               log,
		reportFrequency:   reportFrequency,
		lastReportedBlock: int(config.First) - (int(config.First) % reportFrequency),
	}
}

// progressTracker logs progress every XXX blocks depending on reportFrequency.
// Default is 100_000 blocks. This is mainly used for gathering information about process.
type progressTracker struct {
	extension.NilExtension
	config              *utils.Config
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

func (t *progressTracker) PreRun(_ executor.State, _ *executor.Context) error {
	now := time.Now()
	t.startOfRun = now
	t.startOfLastInterval = now
	return nil
}

// PostTransaction increments number of transactions and saves gas used in last substate.
func (t *progressTracker) PostTransaction(state executor.State, _ *executor.Context) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.overallInfo.numTransactions++
	t.overallInfo.gas += state.Substate.Result.GasUsed

	return nil
}

// PostBlock sends the state to the report goroutine.
// We only care about total number of transactions we can do this here rather in PostTransaction.
func (t *progressTracker) PostBlock(state executor.State, context *executor.Context) error {
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

	disk := utils.GetDirectorySize(t.config.StateDbSrc)
	m := context.State.GetMemoryUsage()

	memory := uint64(0)
	if m != nil {
		memory = m.UsedBytes
	}

	intervalTxRate := float64(info.numTransactions-t.lastIntervalInfo.numTransactions) / interval.Seconds()
	intervalGasRate := float64(info.gas-t.lastIntervalInfo.gas) / interval.Seconds()
	t.lastIntervalInfo = info

	overallTxRate := float64(info.numTransactions) / overall.Seconds()
	overallGasRate := float64(info.gas) / overall.Seconds()

	t.log.Noticef(progressTrackerReportFormat, boundary, memory, disk, intervalTxRate, intervalGasRate, overallTxRate, overallGasRate)

	t.lastReportedBlock = boundary
	t.startOfLastInterval = now

	return nil
}

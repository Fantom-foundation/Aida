package extension

import (
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

const (
	ProgressTrackerDefaultReportFrequency = 100_000 // in blocks
	progressTrackerReportFormat           = "Reached block %d; using ~ %v bytes of memory, ~ %v bytes of disk, last interval rate ~ %v Tx/s, ~ %v Gas/s"
)

func MakeProgressTracker(config *utils.Config, reportFrequency int) executor.Extension {
	if config.Quiet {
		return NilExtension{}
	}

	if reportFrequency == 0 {
		reportFrequency = ProgressTrackerDefaultReportFrequency
	}

	return makeProgressTracker(config, reportFrequency, logger.NewLogger(config.LogLevel, "ProgressTracker"))
}

func makeProgressTracker(config *utils.Config, reportFrequency int, log logger.Logger) *progressTracker {
	return &progressTracker{
		config:           config,
		log:              log,
		reportFrequency:  reportFrequency,
		lastIntervalInfo: new(atomic.Pointer[processInfo]),
	}
}

// progressTracker logs progress every XXX blocks depending on reportFrequency.
// Default is 100_000 blocks. This is mainly used for gathering information about process.
type progressTracker struct {
	NilExtension
	config            *utils.Config
	log               logger.Logger
	reportFrequency   int
	lastReportedBlock int
	start             time.Time
	lastIntervalInfo  *atomic.Pointer[processInfo]
}

type processInfo struct {
	numTransactions uint64
	gas             uint64
}

// PreRun initialises the lastReportedBlock variables for the first time to suppress block
// report at the beginning (in case the user has specified a large enough starting block).
func (t *progressTracker) PreRun(state executor.State) error {
	t.lastReportedBlock = state.Block - (state.Block % t.reportFrequency)

	t.lastIntervalInfo.Store(new(processInfo))

	t.start = time.Now()

	return nil
}

// PostTransaction increments number of transactions and saves gas used in last substate.
func (t *progressTracker) PostTransaction(state executor.State) error {
	info := t.lastIntervalInfo.Load()

	info.numTransactions++
	info.gas += state.Substate.Result.GasUsed

	t.lastIntervalInfo.Store(info)

	return nil
}

// PostBlock sends the state to the report goroutine.
// We only care about total number of transactions we can do this here rather in PostTransaction.
func (t *progressTracker) PostBlock(state executor.State) error {
	boundary := state.Block - (state.Block % t.reportFrequency)

	if state.Block-t.lastReportedBlock < t.reportFrequency {
		return nil
	}

	elapsed := time.Since(t.start)
	info := t.lastIntervalInfo.Load()

	disk := float64(utils.GetDirectorySize(t.config.StateDbSrc))
	m := state.State.GetMemoryUsage()

	var memory float64
	if m == nil {
		memory = 0
	} else {
		memory = float64(m.UsedBytes)
	}

	txRate := float64(info.numTransactions) / elapsed.Seconds()
	gasRate := float64(info.gas) / elapsed.Seconds()

	t.log.Noticef(progressTrackerReportFormat, boundary, disk, memory, txRate, gasRate)

	t.lastReportedBlock = boundary

	return nil
}

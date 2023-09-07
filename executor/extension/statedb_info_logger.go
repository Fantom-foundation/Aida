package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

const (
	StateDbInfoLoggerDefaultReportFrequency = 100_000 // in blocks
	stateDbInfoLoggerReportFormat           = "Reached block %d; disk usage %.2f GiB; memory usage %.2f GiB"
	finalSummaryStateDbInfoReportFormat     = "Total disk usage %.2f GiB; highest memory usage %.2f GiB at block %v"
)

func MakeStateDbInfoLogger(config *utils.Config, reportFrequency int) executor.Extension {
	if config.Quiet {
		return NilExtension{}
	}

	if reportFrequency == 0 {
		reportFrequency = StateDbInfoLoggerDefaultReportFrequency
	}

	return makeStateDbInfoLogger(config, reportFrequency, logger.NewLogger(config.LogLevel, "StateDbInfoLogger"))
}

func makeStateDbInfoLogger(config *utils.Config, reportFrequency int, log logger.Logger) *stateDbInfoLogger {
	return &stateDbInfoLogger{
		config:          config,
		log:             log,
		reportFrequency: reportFrequency,
	}
}

type stateDbInfoLogger struct {
	NilExtension
	config          *utils.Config
	log             logger.Logger
	reportFrequency int
	// we want to know roughly where we had the highest memory usage
	highestMemoryUsage   float64
	highestMemoryBlock   int
	lastBlock            int
	lastBlockInitialized bool // true if the last block got initialized, false otherwise

}

// PostBlock sends the state to the report goroutine.
// We only care about total number of transactions we can do this here rather in PostTransaction.
func (l *stateDbInfoLogger) PostBlock(state executor.State) error {

	// initialise the last-block variables for the first time to suppress block report
	// at the beginning (in case the user has specified a large enough starting block)
	boundary := state.Block - (state.Block % l.reportFrequency)
	if !l.lastBlockInitialized {
		l.lastBlock = boundary
		l.lastBlockInitialized = true
	}

	if !(state.Block-l.lastBlock >= l.reportFrequency) {
		return nil
	}

	disk := float64(utils.GetDirectorySize(l.config.StateDbSrc)) / 1024 / 1024 / 1024
	m := state.State.GetMemoryUsage()

	var memory float64
	if m == nil {
		memory = 0
	} else {
		memory = float64(m.UsedBytes) / 1024 / 1024 / 1024
	}

	l.log.Infof(stateDbInfoLoggerReportFormat, state.Block, disk, memory)

	if memory >= l.highestMemoryUsage {
		l.highestMemoryUsage = memory
		l.highestMemoryBlock = state.Block
	}

	l.lastBlock = boundary

	return nil
}

// PostRun gracefully closes the Extension and awaits the report goroutine correct closure.
func (l *stateDbInfoLogger) PostRun(_ executor.State, _ error) error {
	diskUsage := float64(utils.GetDirectorySize(l.config.StateDbSrc)) / 1024 / 1024 / 1024

	l.log.Noticef(finalSummaryStateDbInfoReportFormat, diskUsage, l.highestMemoryUsage, l.highestMemoryBlock)

	return nil
}

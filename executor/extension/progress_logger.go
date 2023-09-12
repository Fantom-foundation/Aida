package extension

import (
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

const (
	ProgressLoggerDefaultReportFrequency = 15 * time.Second // how often will ticker trigger
	progressLoggerReportFormat           = "Elapsed time: %v; reached block %d; last interval rate ~%.2f Tx/s, ~%.2f Gas/s"
	finalSummaryProgressReportFormat     = "Total elapsed time: %v; reached block %d; total transaction rate ~%.2f Tx/s, ~%.2f Gas/s"
)

// MakeProgressLogger creates progress logger. It logs progress about processor depending on reportFrequency.
// If reportFrequency is 0, it is set to ProgressLoggerDefaultReportFrequency.
func MakeProgressLogger(config *utils.Config, reportFrequency time.Duration) executor.Extension {
	if config.NoHeartbeatLogging {
		return NilExtension{}
	}

	if reportFrequency <= 0 {
		reportFrequency = ProgressLoggerDefaultReportFrequency
	}

	return makeProgressLogger(config, reportFrequency, logger.NewLogger(config.LogLevel, "Progress-Logger"))
}

func makeProgressLogger(config *utils.Config, reportFrequency time.Duration, logger logger.Logger) *progressLogger {
	return &progressLogger{
		config:          config,
		log:             logger,
		inputCh:         make(chan executor.State, config.Workers*10),
		wg:              new(sync.WaitGroup),
		reportFrequency: reportFrequency,
	}
}

// progressLogger logs human-readable information about progress
// in "heartbeat" depending on reportFrequency.
type progressLogger struct {
	NilExtension
	config          *utils.Config
	log             logger.Logger
	inputCh         chan executor.State
	wg              *sync.WaitGroup
	reportFrequency time.Duration
}

// PreRun starts the report goroutine
func (l *progressLogger) PreRun(_ executor.State) error {
	l.wg.Add(1)

	// pass the value for thread safety
	go l.startReport(l.reportFrequency)
	return nil
}

// PostRun gracefully closes the Extension and awaits the report goroutine correct closure.
func (l *progressLogger) PostRun(_ executor.State, _ error) error {
	close(l.inputCh)
	l.wg.Wait()

	return nil
}

func (l *progressLogger) PostTransaction(state executor.State) error {
	l.inputCh <- state
	return nil
}

// startReport runs in own goroutine. It accepts data from Executor from PostBock func.
// It reports current progress everytime we hit the ticker with defaultReportFrequencyInSeconds.
func (l *progressLogger) startReport(reportFrequency time.Duration) {
	start := time.Now()
	lastReport := time.Now()
	ticker := time.NewTicker(reportFrequency)

	var (
		currentBlock                 int
		totalTx, currentIntervalTx   uint64
		totalGas, currentIntervalGas uint64
	)

	defer func() {
		elapsed := time.Since(start)
		txRate := float64(totalTx) / elapsed.Seconds()
		gasRate := float64(totalGas) / elapsed.Seconds()

		l.log.Noticef(finalSummaryProgressReportFormat, elapsed.Round(time.Second), currentBlock, txRate, gasRate)

		l.wg.Done()
	}()

	var (
		in executor.State
		ok bool
	)
	for {
		select {
		case in, ok = <-l.inputCh:
			if !ok {
				return
			}

			if in.Block > currentBlock {
				currentBlock = in.Block
			}

			currentIntervalTx++
			currentIntervalGas += in.Substate.Result.GasUsed

		case now := <-ticker.C:
			elapsed := now.Sub(start)
			txRate := float64(currentIntervalTx) / now.Sub(lastReport).Seconds()
			gasRate := float64(currentIntervalTx) / now.Sub(lastReport).Seconds()

			l.log.Infof(progressLoggerReportFormat, elapsed.Round(1*time.Second), currentBlock, txRate, gasRate)

			lastReport = now
			totalTx += currentIntervalTx
			totalGas += currentIntervalGas

			currentIntervalTx = 0
			currentIntervalGas = 0
		}
	}

}

package extension

import (
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

const (
	ProgressLoggerDefaultReportFrequency       = 15 * time.Second // how often will ticker trigger
	transactionProgressLoggerReportFormat      = "Elapsed time: %v; current block %d; last interval rate ~%.2f Tx/s, ~%.2f MGas/s"
	transactionProgressLoggerFinalReportFormat = "Total elapsed time: %v; last block %d; total transaction rate ~%.2f Tx/s, ~%.2f MGas/s"

	operationProgressLoggerReportFormat      = "Elapsed time: %v; current block %d; last operation rate ~%.2f Op/s"
	operationProgressLoggerFinalReportFormat = "Total elapsed time: %v; last block %d; total operation rate ~%.2f Op/s"
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
func (l *progressLogger) PreRun(executor.State, *executor.Context) error {
	l.wg.Add(1)

	switch l.config.ProgressLoggerType {
	case utils.OperationType:
		go l.startOperationProgressLogger(l.reportFrequency)
	case utils.TransactionType:
		go l.startTransactionProgressLogger(l.reportFrequency)
	}
	return nil
}

// PostRun gracefully closes the Extension and awaits the report goroutine correct closure.
func (l *progressLogger) PostRun(executor.State, *executor.Context, error) error {
	close(l.inputCh)
	l.wg.Wait()

	return nil
}

func (l *progressLogger) PostTransaction(state executor.State, _ *executor.Context) error {
	l.inputCh <- state
	return nil
}

// startTransactionProgressLogger runs in own goroutine. It accepts data from Executor from PostBock func.
// It reports current progress everytime we hit the ticker with defaultReportFrequencyInSeconds.
func (l *progressLogger) startTransactionProgressLogger(reportFrequency time.Duration) {
	defer l.wg.Done()

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

		l.log.Noticef(transactionProgressLoggerFinalReportFormat, elapsed.Round(time.Second), currentBlock, txRate, gasRate/1e6)
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
			gasRate := float64(currentIntervalGas) / now.Sub(lastReport).Seconds()

			l.log.Infof(transactionProgressLoggerReportFormat, elapsed.Round(1*time.Second), currentBlock, txRate, gasRate/1e6)

			lastReport = now
			totalTx += currentIntervalTx
			totalGas += currentIntervalGas

			currentIntervalTx = 0
			currentIntervalGas = 0
		}
	}

}

func (l *progressLogger) startOperationProgressLogger(reportFrequency time.Duration) {
	defer l.wg.Done()

	start := time.Now()
	lastReport := time.Now()
	ticker := time.NewTicker(reportFrequency)

	var (
		currentBlock                               int
		totalOperations, currentIntervalOperations uint64
	)

	defer func() {
		elapsed := time.Since(start)
		opRate := float64(totalOperations) / elapsed.Seconds()

		l.log.Noticef(operationProgressLoggerFinalReportFormat, elapsed.Round(time.Second), currentBlock, opRate)
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

			currentIntervalOperations++
			totalOperations++

		case now := <-ticker.C:
			elapsed := now.Sub(start)
			opRate := float64(currentIntervalOperations) / now.Sub(lastReport).Seconds()

			l.log.Infof(operationProgressLoggerReportFormat, elapsed.Round(1*time.Second), currentBlock, opRate)

			lastReport = now
			currentIntervalOperations = 0
		}
	}

}

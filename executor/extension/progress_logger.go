package extension

import (
	"math/big"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

const (
	ProgressLoggerDefaultReportFrequency = 15 * time.Second // how often will ticker trigger
	progressLoggerReportFormat           = "Elapsed time: %v; reached block %d; last interval rate ~%.2f Tx/s"
	deferProgressLoggerReportFormat      = "Total elapsed time: %v; reached block %d; total transaction rate ~%.2f Tx/s"
)

// MakeProgressLogger creates progress logger. It logs progress about processor depending on reportFrequency.
// If reportFrequency is 0, it is set to ProgressLoggerDefaultReportFrequency.
func MakeProgressLogger(config *utils.Config, reportFrequency time.Duration) executor.Extension {
	if config.Quiet {
		return NilExtension{}
	}

	if reportFrequency == 0 {
		reportFrequency = ProgressLoggerDefaultReportFrequency
	}

	return &progressLogger{
		config:          config,
		log:             logger.NewLogger(config.LogLevel, "Progress-Logger"),
		inputCh:         make(chan executor.State, 10),
		wg:              new(sync.WaitGroup),
		reportFrequency: reportFrequency,
	}
}

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

// PostBlock sends the state to the report goroutine.
// We only care about total number of transactions we can do this here rather in PostTransaction.
func (l *progressLogger) PostBlock(state executor.State) error {
	l.inputCh <- state
	return nil
}

// startReport runs in own goroutine. It accepts data from Executor from PostBock func.
// It reports current progress everytime we hit the ticker with defaultReportFrequencyInSeconds.
func (l *progressLogger) startReport(reportFrequency time.Duration) {
	start := time.Now()
	lastReport := time.Now()
	ticker := time.NewTicker(reportFrequency)

	totalTx := new(big.Int)
	totalBlocks := new(big.Int)
	var lastIntervalTx uint64

	defer func() {
		elapsed := time.Since(start)
		txRate := float64(totalTx.Uint64()) / elapsed.Seconds()

		l.log.Noticef(deferProgressLoggerReportFormat, elapsed.Round(time.Second), totalBlocks.Uint64(), txRate)

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

			// we must do tx + 1 because first tx is actually marked as 0
			lastIntervalTx += uint64(in.Transaction + 1)
			totalBlocks.SetUint64(uint64(in.Block))
		case <-ticker.C:
			elapsed := time.Since(start)
			txRate := float64(lastIntervalTx) / time.Since(lastReport).Seconds()

			// todo add file size and gas rate once StateDb is added to new processor
			l.log.Infof(progressLoggerReportFormat, elapsed.Round(1*time.Second), totalBlocks.Uint64(), txRate)
			lastReport = time.Now()
			totalTx.SetUint64(totalTx.Uint64() + lastIntervalTx)
		}
	}

}

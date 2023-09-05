package extension

import (
	"math/big"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
)

const (
	DefaultReportFrequencyInBlocks = 100_000
	progressReportFormat           = "Elapsed time: %v; reached block %d; last interval rate ~%.2f Tx/s; memory usage ~%.2f GiB; disk usage ~%.2f GiB"
)

// MakeProgressLogger creates progress logger. It logs progress about processor depending on reportFrequency.
// If reportFrequency is 0, it is set to DefaultReportFrequencyInBlocks.
func MakeProgressLogger(config *utils.Config, reportFrequency int) executor.Extension {
	if config.Quiet {
		return NilExtension{}
	}

	if reportFrequency == 0 {
		reportFrequency = DefaultReportFrequencyInBlocks
	}

	return &progressLogger{
		config:          config,
		log:             logger.NewLogger(config.LogLevel, "Progress-Reporter"),
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
	reportFrequency int
}

// PreRun starts the report goroutine
func (l *progressLogger) PreRun(_ executor.State) error {
	l.wg.Add(1)

	// pass the value for thread safety
	go l.startReport(l.reportFrequency, l.config.StateDbSrc)
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
func (l *progressLogger) startReport(reportFrequency int, statePath string) {
	start := time.Now()
	lastReport := time.Now()

	totalTx := new(big.Int)
	totalBlocks := new(big.Int)

	var (
		lastIntervalTx uint64
		in             executor.State
		ok             bool
	)

	defer func() {
		elapsed := time.Since(start)
		txRate := float64(totalTx.Uint64()) / elapsed.Seconds()

		memoryUsage := float64(in.State.GetMemoryUsage().UsedBytes) / 1024 / 1024 / 1024 // convert to GiB
		diskUsage := float64(utils.GetDirectorySize(statePath)) / 1024 / 1024 / 1024     // convert to GiB

		l.log.Infof(progressReportFormat, elapsed.Round(time.Second), totalBlocks.Uint64(), txRate, memoryUsage, diskUsage)

		l.wg.Done()
	}()

	for {
		select {
		case in, ok = <-l.inputCh:
			if !ok {
				return
			}

			// we must do tx + 1 because first tx is actually marked as 0
			lastIntervalTx += uint64(in.Transaction + 1)
			totalBlocks.SetUint64(uint64(in.Block))

			// did we hit the block milestone?
			if in.Block%reportFrequency == 0 {
				l.report(start, lastReport, lastIntervalTx, in.State, statePath, totalBlocks.Uint64())

				// add values to total counter
				totalTx.SetUint64(totalTx.Uint64() + lastIntervalTx)

				// reset interval values
				lastReport = time.Now()
				lastIntervalTx = 0
			}
		}
	}

}

// report calculates the data from Executor and logs info to os.Stdout.
func (l *progressLogger) report(
	start, lastReport time.Time,
	lastIntervalTx uint64,
	state state.StateDB,
	statePath string,
	totalBlocks uint64,
) {
	elapsed := time.Since(start)
	txRate := float64(lastIntervalTx) / time.Since(lastReport).Seconds()

	memoryUsage := float64(state.GetMemoryUsage().UsedBytes) / 1024 / 1024 / 1024 // convert to GiB
	diskUsage := float64(utils.GetDirectorySize(statePath)) / 1024 / 1024 / 1024  // convert to GiB

	// todo add file size and gas rate once StateDb is added to new processor
	l.log.Infof(progressReportFormat,
		elapsed.Round(1*time.Second), totalBlocks, txRate, memoryUsage, diskUsage)
}

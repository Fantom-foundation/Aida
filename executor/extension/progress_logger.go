package extension

import (
	"math/big"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/op/go-logging"
)

const DefaultReportFrequency = 10 * time.Second

// MakeProgressLogger creates progress logger. It logs progress about processor depending on reportFrequency.
// If reportFrequency is 0, it is set to DefaultReportFrequency.
func MakeProgressLogger(config *utils.Config, reportFrequency time.Duration) executor.Extension {
	if config.Quiet {
		return NilExtension{}
	}

	if reportFrequency == 0 {
		reportFrequency = DefaultReportFrequency
	}

	return &progressLogger{
		config:          config,
		log:             logger.NewLogger(config.LogLevel, "Progress-Reporter"),
		inputCh:         make(chan executor.State, 10),
		closeCh:         make(chan any, 1),
		wg:              new(sync.WaitGroup),
		reportFrequency: reportFrequency,
	}
}

type progressLogger struct {
	NilExtension
	config          *utils.Config
	log             *logging.Logger
	inputCh         chan executor.State
	closeCh         chan any
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
	l.close()
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
	ticker := time.NewTicker(reportFrequency)
	start := time.Now()

	totalTx := new(big.Int)
	totalBlocks := new(big.Int)
	var lastIntervalTx uint64

	defer func() {
		elapsed := time.Since(start)
		txRate := float64(totalTx.Uint64()) / elapsed.Seconds()

		close(l.inputCh)

		l.log.Infof("Elapsed time: %v; reached block %d; interval rate ~%.2f Tx/s",
			elapsed.Round(1*time.Second), totalBlocks.Uint64(), txRate)

		l.wg.Done()
	}()

	var in executor.State
	for {
		select {
		case <-l.closeCh:
			return
		case in = <-l.inputCh:
			// we must do tx + 1 because first tx is actually marked as 0
			lastIntervalTx += uint64(in.Transaction + 1)
			totalBlocks.SetUint64(uint64(in.Block))
		case <-ticker.C:
			elapsed := time.Since(start)
			txRate := float64(lastIntervalTx) / reportFrequency.Seconds()

			// todo add file size and gas rate once StateDb is added to new processor
			l.log.Infof("Elapsed time: %v; reached block %d; last interval rate ~%.2f Tx/s",
				elapsed.Round(1*time.Second), totalBlocks.Uint64(), txRate)
		}
	}

}

// close sends signal to the report goroutine to gracefully end
func (l *progressLogger) close() {
	select {
	case <-l.closeCh:
		return
	default:
		close(l.closeCh)
	}
}

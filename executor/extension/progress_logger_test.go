package extension

import (
	"sync"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

const testReportFrequency = 1 * time.Second

func TestProgressLoggerExtension_CorrectClose(t *testing.T) {
	config := &utils.Config{}
	ext := MakeProgressLogger(config, testReportFrequency)

	// start the report thread
	ext.PreRun(executor.State{})

	// signal that the thread was closed correctly
	good := make(chan any, 1)

	go func() {
		// we need a way out in case the logger does not close correctly
		ticker := time.NewTicker(2 * time.Second)
		for {
			select {
			case <-good:
				return
			case <-ticker.C:
				t.Errorf("Logger did not close correctly")
				return
			}
		}
	}()

	ext.PostRun(executor.State{}, nil)
	close(good)
}

func TestProgressLoggerExtension_NoLoggerIsCreatedIfDisabled(t *testing.T) {
	config := &utils.Config{}
	config.Quiet = true
	ext := MakeProgressLogger(config, testReportFrequency)
	if _, ok := ext.(NilExtension); !ok {
		t.Errorf("Logger is enabled although not set in configuration")
	}

}

func TestProgressLoggerExtension_LoggingHappens(t *testing.T) {
	config := &utils.Config{}
	config.Quiet = true

	l, buf := logger.NewTestLogger()

	ext := progressLogger{
		config:          config,
		log:             l,
		inputCh:         make(chan executor.State, 10),
		closeCh:         make(chan any, 1),
		wg:              new(sync.WaitGroup),
		reportFrequency: testReportFrequency,
	}

	ext.PreRun(executor.State{})

	// fill the logger with some data
	ext.PostBlock(executor.State{
		Block:       1,
		Transaction: 0,
	})

	ext.PostBlock(executor.State{
		Block:       1,
		Transaction: 1,
	})

	ext.PostBlock(executor.State{
		Block:       2,
		Transaction: 0,
	})

	ext.PostBlock(executor.State{
		Block:       2,
		Transaction: 1,
	})

	// wait for the ticker to tick
	time.Sleep(2 * time.Second)

	// check if perpetual logging happened
	if buf.String() == "" {
		t.Errorf("Logger did not produce any messages")
	}

	buf.Reset()

	ext.PostRun(executor.State{}, nil)

	// check if defer logging happened
	if buf.String() == "" {
		t.Errorf("Logger did not produce any messages")
	}

}

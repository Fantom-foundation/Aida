package extension

import (
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
)

func TestProgressLoggerExtension_CorrectClose(t *testing.T) {
	config := &utils.Config{}
	ext := MakeProgressLogger(config)

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
	ext := MakeProgressLogger(config)
	if _, ok := ext.(NilExtension); !ok {
		t.Errorf("Logger is enabled although not set in configuration")
	}

}

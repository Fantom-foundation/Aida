package extension

import (
	"fmt"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"go.uber.org/mock/gomock"
)

const testProgressReportFrequency = time.Second

func TestProgressLoggerExtension_CorrectClose(t *testing.T) {
	config := &utils.Config{}
	ext := MakeProgressLogger(config, testProgressReportFrequency)

	// start the report thread
	ext.PreRun(executor.State{})

	// make sure PostRun is not blocking.
	done := make(chan bool)
	go func() {
		ext.PostRun(executor.State{}, nil)
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(time.Second):
		t.Fatalf("PostRun blocked unexpectedly")
	}
}

func TestProgressLoggerExtension_NoLoggerIsCreatedIfDisabled(t *testing.T) {
	config := &utils.Config{}
	config.Quiet = true
	ext := MakeProgressLogger(config, testProgressReportFrequency)
	if _, ok := ext.(NilExtension); !ok {
		t.Errorf("Logger is enabled although not set in configuration")
	}

}

func TestProgressLoggerExtension_LoggingHappens(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	config := &utils.Config{}

	ext := makeProgressLogger(config, testProgressReportFrequency, log)

	ext.PreRun(executor.State{})

	gomock.InOrder(
		// scheduled logging
		log.EXPECT().Infof(progressLoggerReportFormat, gomock.Any(), 1, MatchRate("txRate"), MatchRate("gasRate")).MinTimes(1).MaxTimes(2),
		// defer logging
		log.EXPECT().Noticef(finalSummaryProgressReportFormat, gomock.Any(), 1, MatchRate("txRate"), MatchRate("gasRate")),
	)

	// fill the logger with some data
	ext.PostTransaction(executor.State{
		Block:       1,
		Transaction: 1,
		Substate: &substate.Substate{
			Result: &substate.SubstateResult{
				GasUsed: 1,
			},
		},
	})

	// we must wait for the ticker to tick
	time.Sleep(time.Second)

	ext.PostRun(executor.State{}, nil)
}

// MATCHERS
func MatchRate(name string) gomock.Matcher {
	return matchTxRate{name}
}

type matchTxRate struct {
	name string
}

func (m matchTxRate) Matches(value any) bool {
	txRate, ok := value.(float64)
	return ok && txRate > 0
}

func (m matchTxRate) String() string {
	return fmt.Sprintf("log should have a %v that is larger than 0", m.name)
}

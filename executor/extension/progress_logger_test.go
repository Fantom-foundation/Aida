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
	ext.PreRun(executor.State{}, nil)

	// make sure PostRun is not blocking.
	done := make(chan bool)
	go func() {
		ext.PostRun(executor.State{}, nil, nil)
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
	config.NoHeartbeatLogging = true
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

	ext.PreRun(executor.State{}, nil)

	gomock.InOrder(
		// scheduled logging
		log.EXPECT().Infof(transactionProgressLoggerReportFormat,
			gomock.Any(), 1,
			MatchRate(gomock.All(executor.Gt(0.9), executor.Lt(1.1)), "txRate"),
			MatchRate(gomock.All(executor.Gt(90), executor.Lt(100)), "gasRate"),
		),
		// defer logging
		log.EXPECT().Noticef(transactionProgressLoggerFinalReportFormat,
			gomock.Any(), 1,
			MatchRate(gomock.All(executor.Gt(0.6), executor.Lt(0.7)), "txRate"),
			MatchRate(gomock.All(executor.Gt(60), executor.Lt(70)), "gasRate"),
		),
	)

	// fill the logger with some data
	ext.PostTransaction(executor.State{
		Block:       1,
		Transaction: 1,
		Substate: &substate.Substate{
			Result: &substate.SubstateResult{
				GasUsed: 100_000_000,
			},
		},
	}, nil)

	// we must wait for the ticker to tick
	time.Sleep((3 * testProgressReportFrequency) / 2)

	ext.PostRun(executor.State{}, nil, nil)
}

func TestProgressLoggerExtension_LoggingHappensEvenWhenProgramEndsBeforeTickerTicks(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	config := &utils.Config{}

	// we set large tick rate that does not trigger the ticker
	ext := makeProgressLogger(config, 10*time.Second, log)

	ext.PreRun(executor.State{}, nil)

	log.EXPECT().Noticef(finalSummaryProgressReportFormat,
		gomock.Any(), 1,
		MatchRate(gomock.All(executor.Gt(0.6), executor.Lt(0.7)), "txRate"),
		MatchRate(gomock.All(executor.Gt(60), executor.Lt(70)), "gasRate"),
	)

	// fill the logger with some data
	ext.PostTransaction(executor.State{
		Block:       1,
		Transaction: 1,
		Substate: &substate.Substate{
			Result: &substate.SubstateResult{
				GasUsed: 100_000_000,
			},
		},
	}, nil)

	// wait for data to get into logger
	time.Sleep((3 * testProgressReportFrequency) / 2)

	ext.PostRun(executor.State{}, nil, nil)
}

// MATCHERS
func MatchRate(constraint gomock.Matcher, name string) gomock.Matcher {
	return matchRate{constraint, name}
}

type matchRate struct {
	constraint gomock.Matcher
	name       string
}

func (m matchRate) Matches(value any) bool {
	txRate, ok := value.(float64)
	return ok && m.constraint.Matches(txRate)
}

func (m matchRate) String() string {
	return fmt.Sprintf("log should have a %v that is %v", m.name, m.constraint)
}

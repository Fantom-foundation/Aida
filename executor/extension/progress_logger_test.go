package extension

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

const testProgressReportFrequency = time.Second / 4

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
		log.EXPECT().Infof(MatchFormat(progressLoggerReportFormat), gomock.Any(), 1, MatchTxRate(), float64(0)),
		// defer logging
		log.EXPECT().Noticef(MatchFormat(finalSummaryProgressReportFormat), gomock.Any(), 1, MatchTxRate(), float64(0)),
	)

	// fill the logger with some data
	ext.PostTransaction(executor.State{
		Block:       1,
		Transaction: 1,
	})

	// we must wait for the ticker to tick
	time.Sleep(time.Second / 4)

	ext.PostRun(executor.State{}, nil)
}

// MATCHERS

func MatchFormat(format string) gomock.Matcher {
	return matchFormat{format}
}

type matchFormat struct {
	format string
}

func (m matchFormat) Matches(value any) bool {
	format, ok := value.(string)
	return ok && strings.Compare(m.format, format) == 0
}

func (m matchFormat) String() string {
	return fmt.Sprintf("log format should look like this: %v", m.format)
}

func MatchTxRate() gomock.Matcher {
	return matchTxRate{}
}

type matchTxRate struct {
}

func (m matchTxRate) Matches(value any) bool {
	txRate, ok := value.(float64)
	return ok && txRate > 0
}

func (m matchTxRate) String() string {
	return fmt.Sprintf("log should have a txRate that is larger than 0")
}

package extension

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

const testReportFrequency = 1

func TestProgressLoggerExtension_CorrectClose(t *testing.T) {
	config := &utils.Config{}
	ext := MakeProgressLogger(config, testReportFrequency)

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
	ext := MakeProgressLogger(config, testReportFrequency)
	if _, ok := ext.(NilExtension); !ok {
		t.Errorf("Logger is enabled although not set in configuration")
	}

}

func TestProgressLoggerExtension_LoggingHappens(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	config := &utils.Config{}
	config.Quiet = true

	ext := progressLogger{
		config:          config,
		log:             log,
		inputCh:         make(chan executor.State, 10),
		wg:              new(sync.WaitGroup),
		reportFrequency: testReportFrequency,
	}

	gomock.InOrder(
		db.EXPECT().GetMemoryUsage(),
		log.EXPECT().Infof(MatchFormat(progressReportFormat), gomock.Any(), uint64(1), MatchTxRate(), 0, 0),
		db.EXPECT().GetMemoryUsage(),
		log.EXPECT().Infof(MatchFormat(progressReportFormat), gomock.Any(), uint64(2), MatchTxRate(), 0, 0),
	)

	ext.PreRun(executor.State{})

	// fill the logger with some data
	ext.PostBlock(executor.State{
		Block:       1,
		Transaction: 1,
		State:       db,
	})

	ext.PostBlock(executor.State{
		Block:       2,
		Transaction: 2,
		State:       db,
	})

	// wait a bit until the logger gets the data
	time.Sleep(6 * time.Second)
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

package extension

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"go.uber.org/mock/gomock"
)

const testStateDbInfoFrequency = 2

func TestProgressTrackerExtension_NoLoggerIsCreatedIfDisabled(t *testing.T) {
	config := &utils.Config{}
	config.TrackProgress = false
	ext := MakeProgressTracker(config, testStateDbInfoFrequency)
	if _, ok := ext.(NilExtension); !ok {
		t.Errorf("Logger is enabled although not set in configuration")
	}

}

func TestProgressTrackerExtension_LoggingHappens(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	config := &utils.Config{}
	config.First = 4

	ext := makeProgressTracker(config, testStateDbInfoFrequency, log)

	context := &executor.Context{State: db}

	s := &substate.Substate{
		Result: &substate.SubstateResult{
			Status:  0,
			GasUsed: 100,
		},
	}

	gomock.InOrder(
		db.EXPECT().GetMemoryUsage(),
		log.EXPECT().Noticef(progressTrackerReportFormat,
			6, int64(0), uint64(0),
			MatchRate(gomock.All(executor.Gt(7), executor.Lt(9)), "txRate"),
			MatchRate(gomock.All(executor.Gt(700), executor.Lt(900)), "gasRate"),
			MatchRate(gomock.All(executor.Gt(7), executor.Lt(9)), "txRate"),
			MatchRate(gomock.All(executor.Gt(700), executor.Lt(900)), "gasRate"),
		),
		db.EXPECT().GetMemoryUsage(),
		log.EXPECT().Noticef(progressTrackerReportFormat,
			8, int64(0), uint64(0),
			MatchRate(gomock.All(executor.Gt(1), executor.Lt(2)), "txRate"),
			MatchRate(gomock.All(executor.Gt(180), executor.Lt(220)), "gasRate"),
			MatchRate(gomock.All(executor.Gt(4), executor.Lt(6)), "txRate"),
			MatchRate(gomock.All(executor.Gt(400), executor.Lt(600)), "gasRate"),
		),
	)

	ext.PreRun(executor.State{}, context)

	// first processed block
	ext.PostTransaction(executor.State{Substate: s}, context)
	ext.PostTransaction(executor.State{Substate: s}, context)
	ext.PostBlock(executor.State{
		Block:    5,
		Substate: s,
	}, context)

	time.Sleep(500 * time.Millisecond)

	// second processed block
	ext.PostTransaction(executor.State{Substate: s}, context)
	ext.PostTransaction(executor.State{Substate: s}, context)
	ext.PostBlock(executor.State{
		Block:    6,
		Substate: s,
	}, context)

	time.Sleep(500 * time.Millisecond)

	ext.PostTransaction(executor.State{Substate: s}, context)
	ext.PostBlock(executor.State{
		Block:    8,
		Substate: s,
	}, context)
}

func TestProgressTrackerExtension_FirstLoggingIsIgnored(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	config := &utils.Config{}
	config.First = 4

	ext := makeProgressTracker(config, testStateDbInfoFrequency, log)

	context := &executor.Context{State: db}

	s := &substate.Substate{
		Result: &substate.SubstateResult{
			Status:  0,
			GasUsed: 10,
		},
	}

	ext.PreRun(executor.State{
		Block:       4,
		Transaction: 0,
		Substate:    s,
	}, context)

	ext.PostTransaction(executor.State{
		Block:       4,
		Transaction: 0,
		Substate:    s,
	}, context)
	ext.PostTransaction(executor.State{
		Block:       4,
		Transaction: 1,
		Substate:    s,
	}, context)
	ext.PostBlock(executor.State{
		Block:       5,
		Transaction: 0,
		Substate:    s,
	}, context)
}

func Test_LoggingFormatMatchesRubyScript(t *testing.T) {
	// NOTE: keep this in sync with the pattern used by scripts/run_throughput_eval.rb
	pattern := `Track: block \d+, memory \d+, disk \d+, interval_tx_rate \d+.\d*, interval_gas_rate \d+.\d*, overall_tx_rate \d+.\d*, overall_gas_rate \d+.\d*`
	example := fmt.Sprintf(progressTrackerReportFormat, 1, 2, 3, 4.5, 6.7, 8.9, 0.1)
	if match, err := regexp.Match(pattern, []byte(example)); !match || err != nil {
		t.Errorf("Logging format '%v' does not match required format '%v'; err %v", example, pattern, err)
	}
}

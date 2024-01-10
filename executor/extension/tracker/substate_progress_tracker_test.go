package tracker

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"go.uber.org/mock/gomock"
)

const testStateDbInfoFrequency = 2

func TestSubstateProgressTrackerExtension_NoLoggerIsCreatedIfDisabled(t *testing.T) {
	cfg := &utils.Config{}
	cfg.TrackProgress = false
	ext := MakeSubstateProgressTracker(cfg, testStateDbInfoFrequency)
	if _, ok := ext.(extension.NilExtension[*substate.Substate]); !ok {
		t.Errorf("Logger is enabled although not set in configuration")
	}

}

func TestSubstateProgressTrackerExtension_LoggingHappens(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	cfg := &utils.Config{}
	cfg.First = 4
	dummyStateDbPath := t.TempDir()

	if err := os.WriteFile(dummyStateDbPath+"/dummy.txt", []byte("hello world"), 0x600); err != nil {
		t.Fatalf("failed to prepare disk content")
	}

	ext := makeSubstateProgressTracker(cfg, testStateDbInfoFrequency, log)

	ctx := &executor.Context{State: db, StateDbPath: dummyStateDbPath}

	s := &substate.Substate{
		Result: &substate.SubstateResult{
			Status:  0,
			GasUsed: 100,
		},
	}

	gomock.InOrder(
		db.EXPECT().GetMemoryUsage().Return(&state.MemoryUsage{UsedBytes: 1234}),
		log.EXPECT().Noticef(substateProgressTrackerReportFormat,
			6, uint64(1234), int64(11),
			executor.MatchRate(gomock.All(executor.Gt(3), executor.Lt(5)), "blkRate"),
			executor.MatchRate(gomock.All(executor.Gt(7), executor.Lt(9)), "txRate"),
			executor.MatchRate(gomock.All(executor.Gt(700), executor.Lt(900)), "gasRate"),
			executor.MatchRate(gomock.All(executor.Gt(3), executor.Lt(5)), "blkRate"),
			executor.MatchRate(gomock.All(executor.Gt(7), executor.Lt(9)), "txRate"),
			executor.MatchRate(gomock.All(executor.Gt(700), executor.Lt(900)), "gasRate"),
		),
		db.EXPECT().GetMemoryUsage().Return(&state.MemoryUsage{UsedBytes: 4321}),
		log.EXPECT().Noticef(substateProgressTrackerReportFormat,
			8, uint64(4321), int64(11),
			executor.MatchRate(gomock.All(executor.Gt(3), executor.Lt(5)), "blkRate"),
			executor.MatchRate(gomock.All(executor.Gt(1), executor.Lt(2)), "txRate"),
			executor.MatchRate(gomock.All(executor.Gt(180), executor.Lt(220)), "gasRate"),
			executor.MatchRate(gomock.All(executor.Gt(3), executor.Lt(5)), "blkRate"),
			executor.MatchRate(gomock.All(executor.Gt(4), executor.Lt(6)), "txRate"),
			executor.MatchRate(gomock.All(executor.Gt(400), executor.Lt(600)), "gasRate"),
		),
	)

	ext.PreRun(executor.State[*substate.Substate]{}, ctx)

	// first processed block
	ext.PostTransaction(executor.State[*substate.Substate]{Data: s}, ctx)
	ext.PostTransaction(executor.State[*substate.Substate]{Data: s}, ctx)
	ext.PostBlock(executor.State[*substate.Substate]{
		Block: 5,
		Data:  s,
	}, ctx)

	time.Sleep(500 * time.Millisecond)

	// second processed block
	ext.PostTransaction(executor.State[*substate.Substate]{Data: s}, ctx)
	ext.PostTransaction(executor.State[*substate.Substate]{Data: s}, ctx)
	ext.PostBlock(executor.State[*substate.Substate]{
		Block: 6,
		Data:  s,
	}, ctx)

	time.Sleep(500 * time.Millisecond)

	ext.PostTransaction(executor.State[*substate.Substate]{Data: s}, ctx)
	ext.PostBlock(executor.State[*substate.Substate]{
		Block: 8,
		Data:  s,
	}, ctx)
}

func TestSubstateProgressTrackerExtension_FirstLoggingIsIgnored(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	cfg := &utils.Config{}
	cfg.First = 4

	ext := makeSubstateProgressTracker(cfg, testStateDbInfoFrequency, log)

	ctx := &executor.Context{State: db}

	s := &substate.Substate{
		Result: &substate.SubstateResult{
			Status:  0,
			GasUsed: 10,
		},
	}

	ext.PreRun(executor.State[*substate.Substate]{
		Block:       4,
		Transaction: 0,
		Data:        s,
	}, ctx)

	ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       4,
		Transaction: 0,
		Data:        s,
	}, ctx)
	ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       4,
		Transaction: 1,
		Data:        s,
	}, ctx)
	ext.PostBlock(executor.State[*substate.Substate]{
		Block:       5,
		Transaction: 0,
		Data:        s,
	}, ctx)
}

func Test_LoggingFormatMatchesRubyScript(t *testing.T) {
	// NOTE: keep this in sync with the pattern used by scripts/run_throughput_eval.rb
	pattern := `Track: block \d+, memory \d+, disk \d+, interval_blk_rate \d+.\d*, interval_tx_rate \d+.\d*, interval_gas_rate \d+.\d*, overall_blk_rate \d+.\d*, overall_tx_rate \d+.\d*, overall_gas_rate \d+.\d*`
	example := fmt.Sprintf(substateProgressTrackerReportFormat, 1, 2, 3, 4.5, 6.7, 8.9, 0.1, 2.3, 4.5)
	if match, err := regexp.Match(pattern, []byte(example)); !match || err != nil {
		t.Errorf("Logging format '%v' does not match required format '%v'; err %v", example, pattern, err)
	}
}

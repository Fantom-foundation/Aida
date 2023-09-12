package extension

import (
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

	s := &substate.Substate{
		Result: &substate.SubstateResult{
			Status:  0,
			GasUsed: 100,
		},
	}

	gomock.InOrder(
		// scheduled logging
		db.EXPECT().GetMemoryUsage(),
		log.EXPECT().Noticef(progressTrackerReportFormat,
			6, int64(0), uint64(0),
			MatchRate(gomock.All(executor.Gt(7), executor.Lt(9)), "txRate"),
			MatchRate(gomock.All(executor.Gt(700), executor.Lt(900)), "gasRate"),
		),
	)

	ext.PreRun(executor.State{
		Block:       4,
		Transaction: 0,
		State:       db,
		Substate:    s,
	})

	// first processed block
	ext.PostTransaction(executor.State{
		Block:       4,
		Transaction: 0,
		State:       db,
		Substate:    s,
	})
	ext.PostTransaction(executor.State{
		Block:       4,
		Transaction: 1,
		State:       db,
		Substate:    s,
	})
	ext.PostBlock(executor.State{
		Block:       5,
		Transaction: 0,
		State:       db,
		Substate:    s,
	})

	time.Sleep(500 * time.Millisecond)

	// second processed block
	ext.PostTransaction(executor.State{
		Block:       5,
		Transaction: 0,
		State:       db,
		Substate:    s,
	})
	ext.PostTransaction(executor.State{
		Block:       5,
		Transaction: 1,
		State:       db,
		Substate:    s,
	})
	ext.PostBlock(executor.State{
		Block:       6,
		Transaction: 0,
		State:       db,
		Substate:    s,
	})

}

func TestProgressTrackerExtension_FirstLoggingIsIgnored(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	config := &utils.Config{}
	config.First = 4

	ext := makeProgressTracker(config, testStateDbInfoFrequency, log)

	s := &substate.Substate{
		Result: &substate.SubstateResult{
			Status:  0,
			GasUsed: 10,
		},
	}

	ext.PreRun(executor.State{
		Block:       4,
		Transaction: 0,
		State:       db,
		Substate:    s,
	})

	ext.PostTransaction(executor.State{
		Block:       4,
		Transaction: 0,
		State:       db,
		Substate:    s,
	})
	ext.PostTransaction(executor.State{
		Block:       4,
		Transaction: 1,
		State:       db,
		Substate:    s,
	})
	ext.PostBlock(executor.State{
		Block:       5,
		Transaction: 0,
		State:       db,
		Substate:    s,
	})

}

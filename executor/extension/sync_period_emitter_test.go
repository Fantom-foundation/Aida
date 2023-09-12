package extension

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestSyncPeriodEmitter_Single(t *testing.T) {
	config := &utils.Config{}
	config.SyncPeriodLength = 300
	ext := MakeTestSyncPeriodEmitter(config)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginSyncPeriod(uint64(0)),
		mockStateDB.EXPECT().EndSyncPeriod(),
	)

	state := executor.State{Block: 0}
	context := &executor.Context{State: mockStateDB}
	if err := ext.PreRun(state, context); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}
	if err := ext.PreBlock(state, context); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostRun(state, context, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}
}

func TestSyncPeriodEmitter_MultipleSyncPeriodsSingleBlockLength(t *testing.T) {
	config := &utils.Config{}
	config.SyncPeriodLength = 1
	ext := MakeTestSyncPeriodEmitter(config)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginSyncPeriod(uint64(0)),
		mockStateDB.EXPECT().EndSyncPeriod(),
		mockStateDB.EXPECT().BeginSyncPeriod(uint64(1)),
		mockStateDB.EXPECT().EndSyncPeriod(),
		mockStateDB.EXPECT().BeginSyncPeriod(uint64(2)),
		mockStateDB.EXPECT().EndSyncPeriod(),
	)

	state := executor.State{Block: 0}
	context := &executor.Context{State: mockStateDB}
	if err := ext.PreRun(state, context); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}
	if err := ext.PreBlock(state, context); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	state.Block = 1
	if err := ext.PreBlock(state, context); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	state.Block = 2
	if err := ext.PreBlock(state, context); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostRun(state, context, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}
}

func TestSyncPeriodEmitter_MultipleSyncPeriodsWithoutBlocks(t *testing.T) {
	config := &utils.Config{}
	config.SyncPeriodLength = 2
	ext := MakeTestSyncPeriodEmitter(config)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginSyncPeriod(uint64(0)),
		mockStateDB.EXPECT().EndSyncPeriod(),
		mockStateDB.EXPECT().BeginSyncPeriod(uint64(1)),
		mockStateDB.EXPECT().EndSyncPeriod(),
		mockStateDB.EXPECT().BeginSyncPeriod(uint64(2)),
		mockStateDB.EXPECT().EndSyncPeriod(),
		mockStateDB.EXPECT().BeginSyncPeriod(uint64(3)),
		mockStateDB.EXPECT().EndSyncPeriod(),
	)

	state := executor.State{Block: 0}
	context := &executor.Context{State: mockStateDB}
	if err := ext.PreRun(state, context); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}
	if err := ext.PreBlock(state, context); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	state.Block = 6
	if err := ext.PreBlock(state, context); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostRun(state, context, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}
}

func TestSyncPeriodEmitter_DisabledBecauseOfInvalidSyncPeriodLength(t *testing.T) {
	config := &utils.Config{}
	config.SyncPeriodLength = 0
	ext := MakeTestSyncPeriodEmitter(config)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	state := executor.State{Block: 0}
	context := &executor.Context{State: mockStateDB}
	if err := ext.PreRun(state, context); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}
	if err := ext.PreBlock(state, context); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostRun(state, context, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}
}

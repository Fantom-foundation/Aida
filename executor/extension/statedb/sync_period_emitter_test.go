package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestSyncPeriodEmitter_Single(t *testing.T) {
	cfg := &utils.Config{}
	cfg.SyncPeriodLength = 300
	ext := MakeTestSyncPeriodEmitter[any](cfg)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginSyncPeriod(uint64(0)),
		mockStateDB.EXPECT().EndSyncPeriod(),
	)

	state := executor.State[any]{Block: 0}
	ctx := &executor.Context{State: mockStateDB}
	if err := ext.PreRun(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}
	if err := ext.PreBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}
}

func TestSyncPeriodEmitter_MultipleSyncPeriodsSingleBlockLength(t *testing.T) {
	cfg := &utils.Config{}
	cfg.SyncPeriodLength = 1
	ext := MakeTestSyncPeriodEmitter[any](cfg)

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

	state := executor.State[any]{Block: 0}
	ctx := &executor.Context{State: mockStateDB}
	if err := ext.PreRun(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}
	if err := ext.PreBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	state.Block = 1
	if err := ext.PreBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	state.Block = 2
	if err := ext.PreBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}
}

func TestSyncPeriodEmitter_MultipleSyncPeriodsWithoutBlocks(t *testing.T) {
	cfg := &utils.Config{}
	cfg.SyncPeriodLength = 2
	ext := MakeTestSyncPeriodEmitter[any](cfg)

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

	state := executor.State[any]{Block: 0}
	ctx := &executor.Context{State: mockStateDB}
	if err := ext.PreRun(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}
	if err := ext.PreBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	state.Block = 6
	if err := ext.PreBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}
}

func TestSyncPeriodEmitter_DisabledBecauseOfInvalidSyncPeriodLength(t *testing.T) {
	cfg := &utils.Config{}
	cfg.SyncPeriodLength = 0
	ext := MakeTestSyncPeriodEmitter[any](cfg)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	state := executor.State[any]{Block: 0}
	ctx := &executor.Context{State: mockStateDB}
	if err := ext.PreRun(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}
	if err := ext.PreBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}
}

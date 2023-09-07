package extension

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestSyncPeriod_Single(t *testing.T) {
	config := &utils.Config{}
	config.SyncPeriodLength = 300
	ext := MakeSyncPeriodHook(config)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginSyncPeriod(uint64(0)),
		mockStateDB.EXPECT().EndSyncPeriod(),
	)

	state := executor.State{
		Block: 0,
		State: mockStateDB,
	}
	if err := ext.PreBlock(state); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostRun(state, nil); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
}

func TestSyncPeriod_MultipleSyncPeriodsSingleBlockLength(t *testing.T) {
	config := &utils.Config{}
	config.SyncPeriodLength = 1
	ext := MakeSyncPeriodHook(config)

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

	state := executor.State{
		Block: 0,
		State: mockStateDB,
	}
	if err := ext.PreBlock(state); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	state.Block = 1

	if err := ext.PreBlock(state); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}

	state.Block = 2

	if err := ext.PreBlock(state); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}

	if err := ext.PostRun(state, nil); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
}

func TestSyncPeriod_MultipleSyncPeriodsWithoutBlocks(t *testing.T) {
	config := &utils.Config{}
	config.SyncPeriodLength = 2
	ext := MakeSyncPeriodHook(config)

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

	state := executor.State{
		Block: 0,
		State: mockStateDB,
	}
	if err := ext.PreBlock(state); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}

	state.Block = 6

	if err := ext.PreBlock(state); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}

	if err := ext.PostRun(state, nil); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
}

func TestSyncPeriod_MultipleSyncPeriodsEmpty(t *testing.T) {
	config := &utils.Config{}
	config.SyncPeriodLength = 2
	ext := MakeSyncPeriodHook(config)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder()

	state := executor.State{
		Block: 0,
		State: mockStateDB,
	}

	if err := ext.PostRun(state, nil); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
}

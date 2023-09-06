package extension

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestBlockAndSyncPeriod_SingleBlock(t *testing.T) {
	config := &utils.Config{}
	config.SyncPeriodLength = 300
	ext := MakeBlockBeginner(config)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginSyncPeriod(uint64(0)),
		mockStateDB.EXPECT().BeginBlock(uint64(1)),
		mockStateDB.EXPECT().EndBlock(),
		mockStateDB.EXPECT().EndSyncPeriod(),
	)

	state := executor.State{
		Block: 1,
		State: mockStateDB,
	}
	if err := ext.PreBlock(state); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostBlock(state); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
	if err := ext.PostRun(state, nil); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
}

func TestBlockAndSyncPeriod_MultipleSyncPeriods(t *testing.T) {
	config := &utils.Config{}
	config.SyncPeriodLength = 2
	ext := MakeBlockBeginner(config)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginSyncPeriod(uint64(0)),
		mockStateDB.EXPECT().BeginBlock(uint64(1)),
		mockStateDB.EXPECT().EndBlock(),
		mockStateDB.EXPECT().EndSyncPeriod(),
		mockStateDB.EXPECT().BeginSyncPeriod(uint64(1)),
		mockStateDB.EXPECT().BeginBlock(uint64(2)),
		mockStateDB.EXPECT().EndBlock(),
		mockStateDB.EXPECT().EndSyncPeriod(),
		// mockStateDB.EXPECT().BeginSyncPeriod(uint64(2)),
		// mockStateDB.EXPECT().BeginBlock(uint64(3)),
		// mockStateDB.EXPECT().EndBlock(),
		// mockStateDB.EXPECT().EndSyncPeriod(),
	)

	state := executor.State{
		Block: 1,
		State: mockStateDB,
	}
	if err := ext.PreBlock(state); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostBlock(state); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}

	state.Block = 2

	if err := ext.PreBlock(state); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostBlock(state); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}

	// state.Block = 3

	// if err := ext.PreBlock(state); err != nil {
	// 	t.Fatalf("failed to to run pre-block: %v", err)
	// }
	// if err := ext.PostBlock(state); err != nil {
	// 	t.Fatalf("failed to to run post-block: %v", err)
	// }

	if err := ext.PostRun(state, nil); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
}

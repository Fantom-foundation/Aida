package extension

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"go.uber.org/mock/gomock"
)

func TestBlockEventExtension_SingleBlock(t *testing.T) {
	ext := MakeBlockEventExtension(false)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginBlock(uint64(0)),
		mockStateDB.EXPECT().EndBlock(),
	)

	state := executor.State{
		Block: 0,
		State: mockStateDB,
	}
	if err := ext.PreBlock(state); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostBlock(state); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
}

func TestBlockEventExtension_SkipEndBlocks(t *testing.T) {
	ext := MakeBlockEventExtension(true)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginBlock(uint64(0)),
	)

	state := executor.State{
		Block: 0,
		State: mockStateDB,
	}
	if err := ext.PreBlock(state); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostBlock(state); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
}

func TestBlockEventExtension_MultipleBlocks(t *testing.T) {
	ext := MakeBlockEventExtension(false)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginBlock(uint64(0)),
		mockStateDB.EXPECT().EndBlock(),
		mockStateDB.EXPECT().BeginBlock(uint64(1)),
		mockStateDB.EXPECT().EndBlock(),
		mockStateDB.EXPECT().BeginBlock(uint64(2)),
		mockStateDB.EXPECT().EndBlock(),
	)

	state := executor.State{
		Block: 0,
		State: mockStateDB,
	}
	if err := ext.PreBlock(state); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostBlock(state); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}

	state.Block = 1
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
}

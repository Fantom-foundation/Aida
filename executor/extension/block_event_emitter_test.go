package extension

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"go.uber.org/mock/gomock"
)

func TestBlockEventEmitter_SingleBlock(t *testing.T) {
	ext := MakeBlockEventEmitter()

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginBlock(uint64(0)),
		mockStateDB.EXPECT().EndBlock(),
	)

	state := executor.State{
		Block: 0,
	}
	context := &executor.Context{
		State: mockStateDB,
	}
	if err := ext.PreBlock(state, context); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostBlock(state, context); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
}

func TestBlockEventEmitter_SkipEndBlocks(t *testing.T) {
	ext := MakeBeginOnlyEmitter()

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	mockStateDB.EXPECT().BeginBlock(uint64(0))

	state := executor.State{
		Block: 0,
	}
	context := &executor.Context{
		State: mockStateDB,
	}
	if err := ext.PreBlock(state, context); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostBlock(state, context); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
}

func TestBlockEventEmitter_MultipleBlocks(t *testing.T) {
	ext := MakeBlockEventEmitter()

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
	}
	context := &executor.Context{
		State: mockStateDB,
	}
	if err := ext.PreBlock(state, context); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostBlock(state, context); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}

	state.Block = 1
	if err := ext.PreBlock(state, context); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostBlock(state, context); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}

	state.Block = 2
	if err := ext.PreBlock(state, context); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PostBlock(state, context); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
}

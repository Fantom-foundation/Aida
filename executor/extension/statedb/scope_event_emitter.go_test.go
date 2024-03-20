package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"go.uber.org/mock/gomock"
)

func TestScopeEventEmitter_SingleBlock(t *testing.T) {
	ext := MakeBlockEventEmitter[any]()

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginBlock(uint64(0)),
		mockStateDB.EXPECT().BeginTransaction(uint32(0)),
		mockStateDB.EXPECT().EndTransaction(),
		mockStateDB.EXPECT().EndBlock(),
	)

	state := executor.State[any]{
		Block:       0,
		Transaction: 0,
	}
	ctx := &executor.Context{
		State: mockStateDB,
	}
	if err := ext.PreBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PreTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-tx: %v", err)
	}
	if err := ext.PostTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run post-tx: %v", err)
	}
	if err := ext.PostBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
}

func TestScopeEventEmitter_MultipleBlocks(t *testing.T) {
	ext := MakeBlockEventEmitter[any]()

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginBlock(uint64(0)),
		mockStateDB.EXPECT().BeginTransaction(uint32(0)),
		mockStateDB.EXPECT().EndTransaction(),
		mockStateDB.EXPECT().EndBlock(),
		mockStateDB.EXPECT().BeginBlock(uint64(1)),
		mockStateDB.EXPECT().BeginTransaction(uint32(1)),
		mockStateDB.EXPECT().EndTransaction(),
		mockStateDB.EXPECT().EndBlock(),
		mockStateDB.EXPECT().BeginBlock(uint64(2)),
		mockStateDB.EXPECT().BeginTransaction(uint32(2)),
		mockStateDB.EXPECT().EndTransaction(),
		mockStateDB.EXPECT().EndBlock(),
	)

	state := executor.State[any]{
		Block:       0,
		Transaction: 0,
	}
	ctx := &executor.Context{
		State: mockStateDB,
	}
	if err := ext.PreBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PreTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-tx: %v", err)
	}
	if err := ext.PostTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run post-tx: %v", err)
	}
	if err := ext.PostBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}

	state.Block = 1
	state.Transaction = 1
	if err := ext.PreBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PreTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-tx: %v", err)
	}
	if err := ext.PostTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run post-tx: %v", err)
	}
	if err := ext.PostBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}

	state.Block = 2
	state.Transaction = 2
	if err := ext.PreBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
	if err := ext.PreTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-tx: %v", err)
	}
	if err := ext.PostTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run post-tx: %v", err)
	}
	if err := ext.PostBlock(state, ctx); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}
}

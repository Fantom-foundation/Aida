package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"go.uber.org/mock/gomock"
)

func TestTxGeneratorBlockEventEmitter_SingleBlock(t *testing.T) {
	ext := MakeTxGeneratorBlockEventEmitter[any]()

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().BeginBlock(uint64(0)),
		mockStateDB.EXPECT().BeginTransaction(uint32(0)),
		mockStateDB.EXPECT().EndTransaction(),
		mockStateDB.EXPECT().EndBlock(),
		mockStateDB.EXPECT().BeginBlock(uint64(1)),
		mockStateDB.EXPECT().BeginTransaction(uint32(0)),
		mockStateDB.EXPECT().EndTransaction(),
		mockStateDB.EXPECT().EndBlock(),
	)

	state := executor.State[any]{
		Block: 0,
	}
	ctx := &executor.Context{
		State: mockStateDB,
	}
	if err := ext.PreTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-transaction: %v", err)
	}
	if err := ext.PostTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-transaction: %v", err)
	}

	// increment the block number to make sure the block is ended
	// and the next block is started
	state.Block = 1
	if err := ext.PreTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-transaction: %v", err)
	}

	if err := ext.PostTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-transaction: %v", err)
	}

	// call post run to end the last block
	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}
}

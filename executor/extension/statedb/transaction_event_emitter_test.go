package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"go.uber.org/mock/gomock"
)

func TestTransactionEventEmitter_PreTransactionCallsBeginTx(t *testing.T) {
	ext := MakeTransactionEventEmitter[any]()

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	mockStateDB.EXPECT().BeginTransaction(uint32(0))

	state := executor.State[any]{
		Transaction: 0,
	}
	ctx := &executor.Context{
		State: mockStateDB,
	}
	if err := ext.PreTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-block: %v", err)
	}
}

func TestTransactionEventEmitter_PostTransactionCallsEndTx(t *testing.T) {
	ext := MakeTransactionEventEmitter[any]()

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	mockStateDB.EXPECT().EndTransaction()

	state := executor.State[any]{
		Block:       0,
		Transaction: 0,
	}
	ctx := &executor.Context{
		State: mockStateDB,
	}

	if err := ext.PostTransaction(state, ctx); err != nil {
		t.Fatalf("failed to to run post-block: %v", err)
	}

}

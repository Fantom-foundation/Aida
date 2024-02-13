package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest/state_test"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"go.uber.org/mock/gomock"
)

func TestEthStateBlockEventEmitter_PreTransactionCallsBeginBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	ext := ethStateBlockEventEmitter{}

	db.EXPECT().BeginBlock(uint64(1))

	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: state_test.CreateTestData(t)}
	ctx := &executor.Context{State: db}
	err := ext.PreTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}
}

func TestEthStateBlockEventEmitter_PostTransactionCallsEndBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	ext := ethStateBlockEventEmitter{}

	db.EXPECT().EndBlock()

	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: state_test.CreateTestData(t)}
	ctx := &executor.Context{State: db}
	err := ext.PostTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}
}

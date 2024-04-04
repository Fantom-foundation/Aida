package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"go.uber.org/mock/gomock"
)

func TestEthStateScopeEventEmitter_PreTransactionCallsBeginBlockAndBeginTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	ext := ethStateScopeEventEmitter{}

	db.EXPECT().BeginBlock(uint64(1))
	db.EXPECT().BeginTransaction(uint32(1))

	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: ethtest.CreateTestData(t)}
	ctx := &executor.Context{State: db}
	err := ext.PreTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}
}

func TestEthStateScopeEventEmitter_PostTransactionCallsEndBlockAndEndTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	ext := ethStateScopeEventEmitter{}

	db.EXPECT().EndTransaction()
	db.EXPECT().EndBlock()

	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: ethtest.CreateTestData(t)}
	ctx := &executor.Context{State: db}
	err := ext.PostTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}
}

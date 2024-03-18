package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func Test_ethStateTestDbPrimer_PreTransactionPriming(t *testing.T) {
	cfg := &utils.Config{}
	ext := ethStateTestDbPrimer{cfg: cfg, log: logger.NewLogger(cfg.LogLevel, "EthStatePrimer")}

	testData := ethtest.CreateTestData(t)
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: testData}
	ctx := &executor.Context{}

	mockCtrl := gomock.NewController(t)
	mockState := state.NewMockStateDB(mockCtrl)
	mockLoad := state.NewMockBulkLoad(mockCtrl)

	mockState.EXPECT().StartBulkLoad(uint64(0)).Return(mockLoad, nil)
	for address, account := range testData.Pre {
		mockState.EXPECT().Exist(address).Return(false)
		mockLoad.EXPECT().CreateAccount(address)
		mockLoad.EXPECT().SetBalance(address, account.Balance)
		mockLoad.EXPECT().SetNonce(address, account.Nonce)
		mockLoad.EXPECT().SetCode(address, account.Code)
	}
	mockLoad.EXPECT().Close()

	ctx.State = mockState

	err := ext.PreTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}
}

func Test_ethStateTestDbPrimer_PreTransactionPriming_AlreadyExisting(t *testing.T) {
	cfg := &utils.Config{}
	ext := ethStateTestDbPrimer{cfg: cfg, log: logger.NewLogger(cfg.LogLevel, "EthStatePrimer")}

	testData := ethtest.CreateTestData(t)
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: testData}
	ctx := &executor.Context{}

	mockCtrl := gomock.NewController(t)
	mockState := state.NewMockStateDB(mockCtrl)
	mockLoad := state.NewMockBulkLoad(mockCtrl)

	mockState.EXPECT().StartBulkLoad(uint64(0)).Return(mockLoad)
	for address, account := range testData.Pre {
		mockState.EXPECT().Exist(address).Return(true)
		mockLoad.EXPECT().SetBalance(address, account.Balance)
		mockLoad.EXPECT().SetNonce(address, account.Nonce)
		mockLoad.EXPECT().SetCode(address, account.Code)
	}
	mockLoad.EXPECT().Close()

	ctx.State = mockState

	err := ext.PreTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}
}

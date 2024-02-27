package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest/statetest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func Test_ethStateTestDbPrimer_PreBlockPriming(t *testing.T) {
	cfg := &utils.Config{}
	ext := ethStateTestDbPrimer{cfg: cfg, log: logger.NewLogger(cfg.LogLevel, "EthStatePrimer")}

	testData := statetest.CreateTestData(t)
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: testData}
	ctx := &executor.Context{}

	mockCtrl := gomock.NewController(t)
	mockState := state.NewMockStateDB(mockCtrl)
	mockLoad := state.NewMockBulkLoad(mockCtrl)

	mockState.EXPECT().StartBulkLoad(uint64(0)).Return(mockLoad)
	for address, account := range testData.Pre {
		mockLoad.EXPECT().CreateAccount(address)
		mockLoad.EXPECT().SetBalance(address, account.Balance)
		mockLoad.EXPECT().SetNonce(address, account.Nonce)
		mockLoad.EXPECT().SetCode(address, account.Code)
	}
	mockLoad.EXPECT().Close()

	ctx.State = mockState

	err := ext.PreBlock(st, ctx)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}
}

// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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

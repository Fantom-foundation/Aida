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

package main

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

func TestVmSdb_Eth_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	cfg := &utils.Config{
		First:             2,
		Last:              4,
		ChainID:           utils.MainnetChainID,
		SkipPriming:       true,
		ContinueOnFailure: true,
		LogLevel:          "Critical",
	}

	data := ethtest.CreateTestData(t)

	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: data})
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 2, Data: data})
			return nil
		})

	gomock.InOrder(
		// Tx 1
		db.EXPECT().BeginBlock(uint64(2)),
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetNonce(data.GetMessage().From()).Return(uint64(1)),
		db.EXPECT().GetCodeHash(data.GetMessage().From()).Return(common.HexToHash("0x0")),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(15),
		db.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 1)), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),

		db.EXPECT().BeginBlock(uint64(2)),
		db.EXPECT().BeginTransaction(uint32(2)),
		db.EXPECT().Prepare(gomock.Any(), 2),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetNonce(data.GetMessage().From()).Return(uint64(1)),
		db.EXPECT().GetCodeHash(data.GetMessage().From()).Return(common.HexToHash("0x0")),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(15),
		db.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 2)), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),
	)

	err := runEth(cfg, provider, db, executor.MakeLiveDbTxProcessor(cfg), nil)
	if err != nil {
		errors.Unwrap(err)
		if strings.Contains(err.Error(), "intrinsic gas too low") {
			return
		}
		t.Fatal("run failed")
	}
}

func TestVmSdb_Eth_AllTransactionsAreProcessedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	ext := executor.NewMockExtension[txcontext.TxContext](ctrl)
	processor := executor.NewMockProcessor[txcontext.TxContext](ctrl)
	cfg := &utils.Config{
		First:       2,
		Last:        4,
		ChainID:     utils.MainnetChainID,
		LogLevel:    "Critical",
		SkipPriming: true,
	}

	data := ethtest.CreateTestData(t)

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			// Block 2
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: data})
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 2, Data: data})
			// Block 3
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 3, Transaction: 1, Data: data})
			//// Block 4
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 4, Transaction: utils.PseudoTx, Data: data})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running sequential mode with 1 worker,
	// all block and transactions need to be in order.
	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[txcontext.TxContext](2), gomock.Any()),

		// Block 2
		// Tx 1
		db.EXPECT().BeginBlock(uint64(2)),
		db.EXPECT().BeginTransaction(uint32(1)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),
		// Tx 2
		db.EXPECT().BeginBlock(uint64(2)),
		db.EXPECT().BeginTransaction(uint32(2)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),
		//
		//// Block 3
		db.EXPECT().BeginBlock(uint64(3)),
		db.EXPECT().BeginTransaction(uint32(1)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),
		//
		//// Block 4
		db.EXPECT().BeginBlock(uint64(4)),
		db.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),
		ext.EXPECT().PostRun(executor.AtBlock[txcontext.TxContext](5), gomock.Any(), nil),
	)

	if err := runEth(cfg, provider, db, processor, []executor.Extension[txcontext.TxContext]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmSdb_Eth_ValidationDoesNotFailOnValidTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	cfg := &utils.Config{
		First:       2,
		Last:        4,
		ChainID:     utils.MainnetChainID,
		SkipPriming: true,
		Validate:    true,
		LogLevel:    "Critical",
	}

	data := ethtest.CreateTestData(t)

	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: data})
			return nil
		})

	gomock.InOrder(
		db.EXPECT().Exist(common.HexToAddress("0x1")).Return(true),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().GetNonce(common.HexToAddress("0x1")).Return(uint64(1)),
		db.EXPECT().GetCode(common.HexToAddress("0x1")).Return([]byte{}),
	)
	gomock.InOrder(
		db.EXPECT().Exist(common.HexToAddress("0x2")).Return(true),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(2000)),
		db.EXPECT().GetNonce(common.HexToAddress("0x2")).Return(uint64(2)),
		db.EXPECT().GetCode(common.HexToAddress("0x2")).Return([]byte{}),
	)

	gomock.InOrder(
		// Tx execution
		db.EXPECT().BeginBlock(uint64(2)),
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetNonce(data.GetMessage().From()).Return(uint64(1)),
		db.EXPECT().GetCodeHash(data.GetMessage().From()).Return(common.HexToHash("0x0")),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(15),
		db.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 1)), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		// EndTransaction is not called because execution fails
	)

	err := runEth(cfg, provider, db, executor.MakeLiveDbTxProcessor(cfg), nil)
	if err != nil {
		errors.Unwrap(err)
		if strings.Contains(err.Error(), "intrinsic gas too low") {
			return
		}
		t.Fatal("run failed")
	}
}

func TestVmSdb_Eth_ValidationDoesFailOnInvalidTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	cfg := &utils.Config{
		First:       2,
		Last:        4,
		ChainID:     utils.MainnetChainID,
		SkipPriming: true,
		Validate:    true,
		LogLevel:    "Critical",
	}

	data := ethtest.CreateTestData(t)

	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: data})
			return nil
		})

	// state map contains two accounts, but the validation order of map is not guaranteed
	gomock.InOrder(
		// Tx 1
		// Validation fails on incorrect input so no db events are expected
		// first account has correct data
		db.EXPECT().Exist(common.HexToAddress("0x1")).Return(true),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().GetNonce(common.HexToAddress("0x1")).Return(uint64(1)),
		db.EXPECT().GetCode(common.HexToAddress("0x1")).Return([]byte{}),
	)
	gomock.InOrder(
		// second account has incorrect balance
		db.EXPECT().Exist(common.HexToAddress("0x2")).Return(true),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(9999)), // incorrect balance
		db.EXPECT().GetNonce(common.HexToAddress("0x2")).Return(uint64(2)),
		db.EXPECT().GetCode(common.HexToAddress("0x2")).Return([]byte{}),
	)
	db.EXPECT().BeginBlock(uint64(2))
	db.EXPECT().BeginTransaction(uint32(1))

	err := runEth(cfg, provider, db, executor.MakeLiveDbTxProcessor(cfg), nil)
	if err == nil {
		t.Fatal("run must fail")
	}

	errors.Unwrap(err)
	if !strings.Contains(err.Error(), "pre alloc validation failed") {
		t.Fatalf("unexpected error\ngot: %v\n want: %v", err, "pre alloc validation failed")
	}

}

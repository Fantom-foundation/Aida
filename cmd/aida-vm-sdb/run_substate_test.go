// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/substate"
	substatetypes "github.com/Fantom-foundation/Substate/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
)

func TestVmSdb_Substate_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, false, "")
	cfg.ContinueOnFailure = true
	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			// Block 2
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(emptyTx)})
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 2, Data: substatecontext.NewTxContext(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 3, Transaction: 1, Data: substatecontext.NewTxContext(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 4, Transaction: utils.PseudoTx, Data: substatecontext.NewTxContext(emptyTx)})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	gomock.InOrder(
		// Block 2
		db.EXPECT().BeginBlock(uint64(2)),
		// Tx 1
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().SetTxContext(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any(), tracing.BalanceDecreaseGasBuy),
		db.EXPECT().EndTransaction(),
		// Tx 2
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		db.EXPECT().BeginTransaction(uint32(2)),
		db.EXPECT().SetTxContext(gomock.Any(), 2),
		db.EXPECT().Snapshot().Return(17),
		db.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any(), tracing.BalanceDecreaseGasBuy),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),
		// Block 3
		db.EXPECT().BeginBlock(uint64(3)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(3)),
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().SetTxContext(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(19),
		db.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any(), tracing.BalanceDecreaseGasBuy),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),
		// Pseudo transaction do not use snapshots.
		db.EXPECT().BeginBlock(uint64(4)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(4)),
		db.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),
	)

	processor, err := executor.MakeLiveDbTxProcessor(cfg)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	// since we are working with mock transactions, run logically fails on 'intrinsic gas too low'
	// since this is a test that tests orded of the db events, we can ignore this error
	err = runSubstates(cfg, provider, db, processor, nil, nil)
	if err == nil {
		t.Fatal("run should fail")
	}
}

func TestVmSdb_Substate_AllTransactionsAreProcessedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	ext := executor.NewMockExtension[txcontext.TxContext](ctrl)
	processor := executor.NewMockProcessor[txcontext.TxContext](ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, false, "")
	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			// Block 2
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(emptyTx)})
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 2, Data: substatecontext.NewTxContext(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 3, Transaction: 1, Data: substatecontext.NewTxContext(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 4, Transaction: utils.PseudoTx, Data: substatecontext.NewTxContext(emptyTx)})
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
		ext.EXPECT().PreBlock(executor.AtBlock[txcontext.TxContext](2), gomock.Any()),
		db.EXPECT().BeginBlock(uint64(2)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		db.EXPECT().BeginTransaction(uint32(1)),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		db.EXPECT().EndTransaction(),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		// Tx 2
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		db.EXPECT().BeginTransaction(uint32(2)),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		db.EXPECT().EndTransaction(),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		db.EXPECT().EndBlock(),
		ext.EXPECT().PostBlock(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),

		// Block 3
		ext.EXPECT().PreBlock(executor.AtBlock[txcontext.TxContext](3), gomock.Any()),
		db.EXPECT().BeginBlock(uint64(3)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(3)),
		db.EXPECT().BeginTransaction(uint32(1)),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		db.EXPECT().EndTransaction(),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		db.EXPECT().EndBlock(),
		ext.EXPECT().PostBlock(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),

		// Block 4
		ext.EXPECT().PreBlock(executor.AtBlock[txcontext.TxContext](4), gomock.Any()),
		db.EXPECT().BeginBlock(uint64(4)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(4)),
		db.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		db.EXPECT().EndTransaction(),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		db.EXPECT().EndBlock(),
		ext.EXPECT().PostBlock(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),

		ext.EXPECT().PostRun(executor.AtBlock[txcontext.TxContext](5), gomock.Any(), nil),
	)

	if err := runSubstates(cfg, provider, db, processor, []executor.Extension[txcontext.TxContext]{ext}, nil); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmSdb_Substate_ValidationDoesNotFailOnValidTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, true, "")
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			return consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(testTx)})
		})

	testSender := common.Address(testTx.Message.From)
	testRecipient := common.Address(*testTx.Message.To)

	gasCosts := new(uint256.Int).Mul(uint256.NewInt(uint64(testTx.Message.Gas)), uint256.MustFromBig(testTx.Message.GasPrice))
	transferValue := uint256.MustFromBig(testTx.Message.Value)

	gomock.InOrder(
		db.EXPECT().BeginBlock(uint64(2)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		db.EXPECT().BeginTransaction(uint32(1)),

		// we return correct expected data so tx does not fail
		// Pre-check and Gas buying
		db.EXPECT().SetTxContext(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetBalance(testSender).Return(gasCosts),
		db.EXPECT().SubBalance(testSender, gasCosts, tracing.BalanceDecreaseGasBuy),
		db.EXPECT().Prepare(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()),
		db.EXPECT().GetNonce(testSender).Return(uint64(1)),
		db.EXPECT().SetNonce(testSender, uint64(2)),
		db.EXPECT().GetBalance(testSender).Return(new(uint256.Int).SetUint64(1)),

		// Actual contract call
		db.EXPECT().Snapshot().Return(16),
		db.EXPECT().Exist(testRecipient).Return(true),
		db.EXPECT().SubBalance(testSender, transferValue, tracing.BalanceChangeTransfer),
		db.EXPECT().AddBalance(testRecipient, transferValue, tracing.BalanceChangeTransfer),
		db.EXPECT().GetCode(testRecipient).Return([]byte{}),
		db.EXPECT().AddBalance(testSender, gomock.Any(), tracing.BalanceIncreaseGasReturn),

		// Post-transaction operations
		db.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 1)), uint64(2), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),
	)

	db.EXPECT().Witness().AnyTimes()
	db.EXPECT().GetRefund().Return(uint64(0)).AnyTimes()

	processor, err := executor.MakeLiveDbTxProcessor(cfg)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	err = runSubstates(cfg, provider, db, processor, nil, nil)
	if err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmSdb_Substate_ValidationFailsOnInvalidTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, true, "")
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			return consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(testTx)})
		})

	testSender := common.Address(testTx.Message.From)

	gomock.InOrder(
		db.EXPECT().BeginBlock(uint64(2)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		db.EXPECT().BeginTransaction(uint32(1)),

		// to make the transaction fail, wo do not provide enough balance to pay for the gas
		// Pre-check and Gas buying
		db.EXPECT().SetTxContext(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetBalance(testSender).Return(uint256.NewInt(0)), // < this is not enough for the gas
		// EndTransaction does not get called because validation fails
	)

	processor, err := executor.MakeLiveDbTxProcessor(cfg)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	err = runSubstates(cfg, provider, db, processor, nil, nil)
	if err == nil {
		t.Errorf("validation must fail")
	}

	expectedErr := "insufficient funds for gas * price + value"
	returnedErr := strings.TrimSpace(err.Error())

	if !strings.Contains(returnedErr, expectedErr) {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

// emptyTx is a dummy substate that will be processed without crashing.
var emptyTx = &substate.Substate{
	Env: &substate.Env{
		GasLimit: 100_000_000,
	},
	Message: &substate.Message{
		GasPrice:  big.NewInt(12),
		Gas:       1,
		GasFeeCap: big.NewInt(1_000_000),
		GasTipCap: big.NewInt(1_000_000),
	},
	Result: &substate.Result{
		GasUsed: 1,
	},
}

// testTx is a dummy substate used for testing validation.
var testTx = &substate.Substate{
	Env: &substate.Env{
		GasLimit: 100_000_000,
	},
	Message: &substate.Message{
		From:      substatetypes.Address{0x01},
		To:        &substatetypes.Address{0x02},
		GasPrice:  big.NewInt(12),
		Value:     big.NewInt(1),
		Gas:       1_000_000,
		GasFeeCap: big.NewInt(1_000_000),
		GasTipCap: big.NewInt(1_000_000),
	},
	Result: &substate.Result{
		Status:  1,
		GasUsed: 118900,
	},
}

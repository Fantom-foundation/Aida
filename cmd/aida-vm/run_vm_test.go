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

var testingAddress = common.Address{1}

func TestVm_AllDbEventsAreIssuedInOrder_Sequential(t *testing.T) {
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

	// The expectation is that all of those  transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running sequential mode with 1 worker,
	// they all need to be in order.
	gomock.InOrder(
		// Block 2
		// Tx 1
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().SetTxContext(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		db.EXPECT().RevertToSnapshot(15),
		db.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 1)), uint64(2), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		db.EXPECT().EndTransaction(),
		// Tx 2
		db.EXPECT().BeginTransaction(uint32(2)),
		db.EXPECT().SetTxContext(gomock.Any(), 2),
		db.EXPECT().Snapshot().Return(17),
		db.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		db.EXPECT().RevertToSnapshot(17),
		db.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 2)), uint64(2), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		db.EXPECT().EndTransaction(),
		// Block 3
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().SetTxContext(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(19),
		db.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		db.EXPECT().RevertToSnapshot(19),
		db.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 3, 1)), uint64(3), common.HexToHash(fmt.Sprintf("0x%016d", 3))),
		db.EXPECT().EndTransaction(),
		// Pseudo Tx
		db.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		db.EXPECT().EndTransaction(),
	)

	processor, err := executor.MakeLiveDbTxProcessor(cfg)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	run(cfg, provider, db, processor, nil)
}

func TestVm_AllTransactionsAreProcessedInOrder_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	processor := executor.NewMockProcessor[txcontext.TxContext](ctrl)
	ext := executor.NewMockExtension[txcontext.TxContext](ctrl)

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

	// The expectation is that all of those transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running sequential mode with 1 worker,
	// all blocks and transactions need to be in order.

	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[txcontext.TxContext](2), gomock.Any()),

		// Block 2
		// Tx 1
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		// Tx 2
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),

		// Block 3
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),

		// Block 4
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),

		ext.EXPECT().PostRun(executor.AtBlock[txcontext.TxContext](5), gomock.Any(), nil),
	)

	run(cfg, provider, nil, processor, []executor.Extension[txcontext.TxContext]{ext})
}

func TestVm_AllTransactionsAreProcessedInOrder_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	processor := executor.NewMockProcessor[txcontext.TxContext](ctrl)
	ext := executor.NewMockExtension[txcontext.TxContext](ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, false, "")
	cfg.ContinueOnFailure = true
	cfg.Workers = 2
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

	pre := ext.EXPECT().PreRun(executor.AtBlock[txcontext.TxContext](2), gomock.Any())
	post := ext.EXPECT().PostRun(executor.AtBlock[txcontext.TxContext](5), gomock.Any(), nil)

	// The expectation is that all of those transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running parallel mode with multiple workers tx
	// order does not have to be preserved.

	// Block 2
	// Tx 1
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		post,
	)

	// Tx 2
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		post,
	)

	// Block 3
	// Tx 1
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		post,
	)

	// Block 4
	// Tx 1
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		post,
	)

	if err := run(cfg, provider, nil, processor, []executor.Extension[txcontext.TxContext]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_ValidationDoesNotFailOnValidTransaction_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, true, "")
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			return consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(testTx)})
		})

	gomock.InOrder(
		// we return correct expected data so tx does not fail
		db.EXPECT().Exist(testingAddress).Return(true),
		db.EXPECT().GetBalance(testingAddress).Return(new(uint256.Int).SetUint64(1)),
		db.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		db.EXPECT().GetCode(testingAddress).Return([]byte{}),

		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().SetTxContext(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any(), tracing.BalanceDecreaseGasBuy),
		db.EXPECT().RevertToSnapshot(15),
		db.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 1)), uint64(2), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
	)

	processor, err := executor.MakeLiveDbTxProcessor(cfg)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	// run fails but not on validation
	err = run(cfg, provider, db, processor, nil)
	if err == nil {
		t.Fatal("run must fail")
	}

	// we expected error with low gas, which means the validation passed
	expectedErr := strings.TrimSpace("intrinsic gas too low: have 0, want 53000")
	returnedErr := strings.TrimSpace(err.Error())

	if !strings.Contains(returnedErr, expectedErr) {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

func TestVmAdb_ValidationDoesNotFailOnValidTransaction_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, true, "")
	cfg.Workers = 2
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			return consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(testTx)})
		})

	gomock.InOrder(
		// we return correct expected data so tx does not fail
		db.EXPECT().Exist(testingAddress).Return(true),
		db.EXPECT().GetBalance(testingAddress).Return(new(uint256.Int).SetUint64(1)),
		db.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		db.EXPECT().GetCode(testingAddress).Return([]byte{}),

		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().SetTxContext(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any(), tracing.BalanceDecreaseGasBuy),
		db.EXPECT().RevertToSnapshot(15),
		db.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 1)), uint64(2), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
	)

	processor, err := executor.MakeLiveDbTxProcessor(cfg)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	// run fails but not on validation
	err = run(cfg, provider, db, processor, nil)
	if err == nil {
		t.Fatal("run must fail")
	}

	// we expected error with low gas, which means the validation passed
	expectedErr := strings.TrimSpace("intrinsic gas too low: have 0, want 53000")
	returnedErr := strings.TrimSpace(err.Error())

	if !strings.Contains(returnedErr, expectedErr) {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

func TestVm_ValidationFailsOnInvalidTransaction_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, true, "")
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			return consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().Exist(testingAddress).Return(false), // address does not exist
		db.EXPECT().GetBalance(testingAddress).Return(new(uint256.Int).SetUint64(1)),
		db.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		db.EXPECT().GetCode(testingAddress).Return([]byte{}),
		db.EXPECT().BeginTransaction(uint32(1)),
	)

	processor, err := executor.MakeLiveDbTxProcessor(cfg)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	err = run(cfg, provider, db, processor, nil)
	if err == nil {
		t.Errorf("validation must fail")
	}

	expectedErr := strings.TrimSpace("live-db-validator err:\nblock 2 tx 1\n world-state input is not contained in the state-db\n   Account 0x0100000000000000000000000000000000000000 does not exist")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

func TestVm_ValidationFailsOnInvalidTransaction_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, true, "")
	cfg.Workers = 2
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			return consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().Exist(testingAddress).Return(false), // address does not exist
		db.EXPECT().GetBalance(testingAddress).Return(new(uint256.Int).SetUint64(1)),
		db.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		db.EXPECT().GetCode(testingAddress).Return([]byte{}),
		db.EXPECT().BeginTransaction(uint32(1)),
	)

	processor, err := executor.MakeLiveDbTxProcessor(cfg)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	err = run(cfg, provider, db, processor, nil)
	if err == nil {
		t.Fatal("validation must fail")
	}

	expectedErr := strings.TrimSpace("live-db-validator err:\nblock 2 tx 1\n world-state input is not contained in the state-db\n   Account 0x0100000000000000000000000000000000000000 does not exist")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

// emptyTx is a dummy substate that will be processed without crashing.
var emptyTx = &substate.Substate{
	Env: &substate.Env{},
	Message: &substate.Message{
		Gas:      10000,
		GasPrice: big.NewInt(0),
		Value:    big.NewInt(1),
	},
	Result: &substate.Result{
		GasUsed: 1,
	},
}

// testTx is a dummy substate used for testing validation.
var testTx = &substate.Substate{
	InputSubstate: substate.WorldState{substatetypes.Address(testingAddress): substate.NewAccount(1, new(big.Int).SetUint64(1), []byte{})},
	Env:           &substate.Env{},
	Message: &substate.Message{
		GasPrice: big.NewInt(12),
		Value:    big.NewInt(1),
	},
	Result: &substate.Result{
		GasUsed: 1,
	},
}

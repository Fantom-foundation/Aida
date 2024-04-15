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

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

var testingAddress = common.Address{1}

func TestVmSdb_Substate_AllDbEventsAreIssuedInOrder(t *testing.T) {
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
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(15),
		db.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 1)), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		db.EXPECT().EndTransaction(),
		// Tx 2
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		db.EXPECT().BeginTransaction(uint32(2)),
		db.EXPECT().Prepare(gomock.Any(), 2),
		db.EXPECT().Snapshot().Return(17),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(17),
		db.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 2)), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),
		// Block 3
		db.EXPECT().BeginBlock(uint64(3)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(3)),
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(19),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(19),
		db.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 3, 1)), common.HexToHash(fmt.Sprintf("0x%016d", 3))),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),
		// Pseudo transaction do not use snapshots.
		db.EXPECT().BeginBlock(uint64(4)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(4)),
		db.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),
	)

	// since we are working with mock transactions, run logically fails on 'intrinsic gas too low'
	// since this is a test that tests orded of the db events, we can ignore this error
	err := runSubstates(cfg, provider, db, executor.MakeLiveDbTxProcessor(cfg), nil)
	if err != nil {
		errors.Unwrap(err)
		if strings.Contains(err.Error(), "intrinsic gas too low") {
			return
		}
		t.Fatalf("run failed; %v", err)
	}
}

func TestVmSdb_Substate_AllTransactionsAreProcessedInOrder(t *testing.T) {
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

	if err := runSubstates(cfg, provider, db, processor, []executor.Extension[txcontext.TxContext]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmSdb_Substate_ValidationDoesNotFailOnValidTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	cfg := &utils.Config{
		First:           2,
		Last:            4,
		ChainID:         utils.MainnetChainID,
		ValidateTxState: true,
		SkipPriming:     true,
	}

	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			return consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().BeginBlock(uint64(2)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		db.EXPECT().BeginTransaction(uint32(1)),

		// we return correct expected data so tx does not fail
		db.EXPECT().Exist(testingAddress).Return(true),
		db.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		db.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		db.EXPECT().GetCode(testingAddress).Return([]byte{}),
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(15),
		db.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 1)), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		// EndTransaction does not get called because execution fails
	)

	// run fails but not on validation
	err := runSubstates(cfg, provider, db, executor.MakeLiveDbTxProcessor(cfg), nil)
	if err == nil {
		t.Errorf("run must fail")
	}

	expectedErr := strings.TrimSpace("block: 2 transaction: 1\nintrinsic gas too low: have 0, want 53000")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

func TestVmSdb_Substate_ValidationFailsOnInvalidTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	cfg := &utils.Config{
		First:           2,
		Last:            4,
		ChainID:         utils.MainnetChainID,
		ValidateTxState: true,
		SkipPriming:     true,
	}

	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			return consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().BeginBlock(uint64(2)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Exist(testingAddress).Return(false), // address does not exist
		db.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		db.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		db.EXPECT().GetCode(testingAddress).Return([]byte{}),
		// EndTransaction does not get called because validation fails
	)

	err := runSubstates(cfg, provider, db, executor.MakeLiveDbTxProcessor(cfg), nil)
	if err == nil {
		t.Errorf("validation must fail")
	}

	expectedErr := strings.TrimSpace("live-db-validator err:\nblock 2 tx 1\n world-state input is not contained in the state-db\n   Account 0x0100000000000000000000000000000000000000 does not exist")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}

}

// emptyTx is a dummy substate that will be processed without crashing.
var emptyTx = &substate.Substate{
	Env: &substate.SubstateEnv{},
	Message: &substate.SubstateMessage{
		GasPrice: big.NewInt(12),
	},
	Result: &substate.SubstateResult{
		GasUsed: 1,
	},
}

// testTx is a dummy substate used for testing validation.
var testTx = &substate.Substate{
	InputAlloc: substate.SubstateAlloc{testingAddress: substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{})},
	Env:        &substate.SubstateEnv{},
	Message: &substate.SubstateMessage{
		GasPrice: big.NewInt(12),
	},
	Result: &substate.SubstateResult{
		GasUsed: 1,
	},
}

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
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
)

var testingAddress = common.Address{1}

func TestVmAdb_AllDbEventsAreIssuedInOrder_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	archiveBlockOne := state.NewMockNonCommittableStateDB(ctrl)
	archiveBlockTwo := state.NewMockNonCommittableStateDB(ctrl)
	archiveBlockThree := state.NewMockNonCommittableStateDB(ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, false)
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
	// Since we are running sequential mode with 1 worker, they all need to be in order.
	gomock.InOrder(
		// Block 2
		// Tx 1
		db.EXPECT().GetArchiveState(uint64(1)).Return(archiveBlockOne, nil),
		archiveBlockOne.EXPECT().BeginTransaction(uint32(1)),
		archiveBlockOne.EXPECT().SetTxContext(gomock.Any(), 1),
		archiveBlockOne.EXPECT().Snapshot().Return(15),
		archiveBlockOne.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		archiveBlockOne.EXPECT().SubBalance(gomock.Any(), gomock.Any(), gomock.Any()),
		archiveBlockOne.EXPECT().RevertToSnapshot(15),
		archiveBlockOne.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 1)), uint64(2), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		archiveBlockOne.EXPECT().EndTransaction(),
		// Tx 2
		archiveBlockOne.EXPECT().BeginTransaction(uint32(2)),
		archiveBlockOne.EXPECT().SetTxContext(gomock.Any(), 2),
		archiveBlockOne.EXPECT().Snapshot().Return(16),
		archiveBlockOne.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		archiveBlockOne.EXPECT().SubBalance(gomock.Any(), gomock.Any(), gomock.Any()),
		archiveBlockOne.EXPECT().RevertToSnapshot(16),
		archiveBlockOne.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 2)), uint64(2), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		archiveBlockOne.EXPECT().EndTransaction(),
		archiveBlockOne.EXPECT().Release(),
		// Block 3
		db.EXPECT().GetArchiveState(uint64(2)).Return(archiveBlockTwo, nil),
		archiveBlockTwo.EXPECT().BeginTransaction(uint32(1)),
		archiveBlockTwo.EXPECT().SetTxContext(gomock.Any(), 1),
		archiveBlockTwo.EXPECT().Snapshot().Return(17),
		archiveBlockTwo.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		archiveBlockTwo.EXPECT().SubBalance(gomock.Any(), gomock.Any(), gomock.Any()),
		archiveBlockTwo.EXPECT().RevertToSnapshot(17),
		archiveBlockTwo.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 3, 1)), uint64(3), common.HexToHash(fmt.Sprintf("0x%016d", 3))),
		archiveBlockTwo.EXPECT().EndTransaction(),
		archiveBlockTwo.EXPECT().Release(),
		// Block 4
		// Pseudo transaction do not use snapshots.
		db.EXPECT().GetArchiveState(uint64(3)).Return(archiveBlockThree, nil),
		archiveBlockThree.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		archiveBlockThree.EXPECT().EndTransaction(),
		archiveBlockThree.EXPECT().Release(),
	)

	// since we are working with mock transactions, run logically fails on 'intrinsic gas too low'
	// since this is a test that tests orded of the db events, we can ignore this error
	err := run(cfg, provider, db, executor.MakeArchiveDbTxProcessor(cfg), nil)
	if err != nil {
		if strings.Contains(err.Error(), "intrinsic gas too low") {
			return
		}
		t.Fatal("run failed")
	}
}

func TestVmAdb_AllDbEventsAreIssuedInOrder_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	archiveBlockOne := state.NewMockNonCommittableStateDB(ctrl)
	archiveBlockTwo := state.NewMockNonCommittableStateDB(ctrl)
	archiveBlockThree := state.NewMockNonCommittableStateDB(ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, false)
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

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running parallel mode with multiple workers,
	// block order does not have to preserved, only transaction order matters.

	// Block 2
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archiveBlockOne, nil),
		// Tx 1
		archiveBlockOne.EXPECT().BeginTransaction(uint32(1)),
		archiveBlockOne.EXPECT().SetTxContext(gomock.Any(), 1),
		archiveBlockOne.EXPECT().Snapshot().Return(15),
		archiveBlockOne.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		archiveBlockOne.EXPECT().SubBalance(gomock.Any(), gomock.Any(), gomock.Any()),
		archiveBlockOne.EXPECT().RevertToSnapshot(15),
		archiveBlockOne.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 1)), uint64(2), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		archiveBlockOne.EXPECT().EndTransaction(),
		// Tx 2
		archiveBlockOne.EXPECT().BeginTransaction(uint32(2)),
		archiveBlockOne.EXPECT().SetTxContext(gomock.Any(), 2),
		archiveBlockOne.EXPECT().Snapshot().Return(19),
		archiveBlockOne.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		archiveBlockOne.EXPECT().SubBalance(gomock.Any(), gomock.Any(), gomock.Any()),
		archiveBlockOne.EXPECT().RevertToSnapshot(19),
		archiveBlockOne.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 2)), uint64(2), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		archiveBlockOne.EXPECT().EndTransaction(),

		archiveBlockOne.EXPECT().Release(),
	)
	// Block 3
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(2)).Return(archiveBlockTwo, nil),
		archiveBlockTwo.EXPECT().BeginTransaction(uint32(1)),
		archiveBlockTwo.EXPECT().SetTxContext(gomock.Any(), 1),
		archiveBlockTwo.EXPECT().Snapshot().Return(17),
		archiveBlockTwo.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		archiveBlockTwo.EXPECT().SubBalance(gomock.Any(), gomock.Any(), gomock.Any()),
		archiveBlockTwo.EXPECT().RevertToSnapshot(17),
		archiveBlockTwo.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 3, 1)), uint64(3), common.HexToHash(fmt.Sprintf("0x%016d", 3))),
		archiveBlockTwo.EXPECT().EndTransaction(),
		archiveBlockTwo.EXPECT().Release(),
	)

	// Block 4
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(3)).Return(archiveBlockThree, nil),
		// Pseudo transaction do not use snapshots.
		archiveBlockThree.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		archiveBlockThree.EXPECT().EndTransaction(),
		archiveBlockThree.EXPECT().Release(),
	)

	// since we are working with mock transactions, run logically fails on 'intrinsic gas too low'
	// since this is a test that tests orded of the db events, we can ignore this error
	err := run(cfg, provider, db, executor.MakeArchiveDbTxProcessor(cfg), nil)
	if err != nil {
		if strings.Contains(err.Error(), "intrinsic gas too low") {
			return
		}
		t.Fatal("run failed")
	}
}

func TestVmAdb_AllTransactionsAreProcessedInOrder_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)
	ext := executor.NewMockExtension[txcontext.TxContext](ctrl)
	processor := executor.NewMockProcessor[txcontext.TxContext](ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, false)
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
	// all blocks and transactions need to be in order.

	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[txcontext.TxContext](2), gomock.Any()),

		// Block 2
		// Tx 1
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		ext.EXPECT().PreBlock(executor.AtBlock[txcontext.TxContext](2), gomock.Any()),
		archive.EXPECT().BeginTransaction(uint32(1)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		archive.EXPECT().EndTransaction(),
		// Tx 2
		archive.EXPECT().BeginTransaction(uint32(2)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		archive.EXPECT().EndTransaction(),
		ext.EXPECT().PostBlock(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		archive.EXPECT().Release(),

		// Block 3
		db.EXPECT().GetArchiveState(uint64(2)).Return(archive, nil),
		ext.EXPECT().PreBlock(executor.AtBlock[txcontext.TxContext](3), gomock.Any()),
		archive.EXPECT().BeginTransaction(uint32(1)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		archive.EXPECT().EndTransaction(),
		ext.EXPECT().PostBlock(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		archive.EXPECT().Release(),

		// Block 4
		db.EXPECT().GetArchiveState(uint64(3)).Return(archive, nil),
		ext.EXPECT().PreBlock(executor.AtBlock[txcontext.TxContext](4), gomock.Any()),
		archive.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		archive.EXPECT().EndTransaction(),
		ext.EXPECT().PostBlock(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		archive.EXPECT().Release(),

		ext.EXPECT().PostRun(executor.AtBlock[txcontext.TxContext](5), gomock.Any(), nil),
	)

	if err := run(cfg, provider, db, processor, []executor.Extension[txcontext.TxContext]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_AllTransactionsAreProcessed_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	archiveBlk2 := state.NewMockNonCommittableStateDB(ctrl)
	archiveBlk3 := state.NewMockNonCommittableStateDB(ctrl)
	archiveBlk4 := state.NewMockNonCommittableStateDB(ctrl)
	ext := executor.NewMockExtension[txcontext.TxContext](ctrl)
	processor := executor.NewMockProcessor[txcontext.TxContext](ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, false)
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

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running parallel mode with multiple workers block
	// order does not have to be preserved, only transaction order matters.
	ext.EXPECT().PreRun(executor.AtBlock[txcontext.TxContext](2), gomock.Any())

	// Block 2
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archiveBlk2, nil),
		ext.EXPECT().PreBlock(executor.AtBlock[txcontext.TxContext](2), gomock.Any()),
		// Tx 1
		archiveBlk2.EXPECT().BeginTransaction(uint32(1)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		archiveBlk2.EXPECT().EndTransaction(),
		// Tx 2
		archiveBlk2.EXPECT().BeginTransaction(uint32(2)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		archiveBlk2.EXPECT().EndTransaction(),
		ext.EXPECT().PostBlock(executor.AtBlock[txcontext.TxContext](2), gomock.Any()),
		archiveBlk2.EXPECT().Release(),
	)

	// Block 3
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(2)).Return(archiveBlk3, nil),
		ext.EXPECT().PreBlock(executor.AtBlock[txcontext.TxContext](3), gomock.Any()),
		archiveBlk3.EXPECT().BeginTransaction(uint32(1)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		archiveBlk3.EXPECT().EndTransaction(),
		ext.EXPECT().PostBlock(executor.AtBlock[txcontext.TxContext](3), gomock.Any()),
		archiveBlk3.EXPECT().Release(),
	)

	// Block 4
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(3)).Return(archiveBlk4, nil),
		ext.EXPECT().PreBlock(executor.AtBlock[txcontext.TxContext](4), gomock.Any()),
		archiveBlk4.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
		archiveBlk4.EXPECT().EndTransaction(),
		ext.EXPECT().PostBlock(executor.AtBlock[txcontext.TxContext](4), gomock.Any()),
		archiveBlk4.EXPECT().Release(),
	)

	ext.EXPECT().PostRun(executor.AtBlock[txcontext.TxContext](5), gomock.Any(), nil)

	if err := run(cfg, provider, db, processor, []executor.Extension[txcontext.TxContext]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_ValidationDoesNotFailOnValidTransaction_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, true)
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			return consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		// we return correct expected data so tx does not fail
		archive.EXPECT().BeginTransaction(uint32(1)),
		archive.EXPECT().Exist(testingAddress).Return(true),
		archive.EXPECT().GetBalance(testingAddress).Return(new(uint256.Int).SetUint64(1)),
		archive.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		archive.EXPECT().GetCode(testingAddress).Return([]byte{}),

		archive.EXPECT().SetTxContext(gomock.Any(), 1),
		archive.EXPECT().Snapshot().Return(15),
		archive.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		archive.EXPECT().SubBalance(gomock.Any(), gomock.Any(), gomock.Any()),
		archive.EXPECT().RevertToSnapshot(15),
		archive.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 1)), uint64(2), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		// end transaction is not called because we expect fail
	)

	// run fails but not on validation
	err := run(cfg, provider, db, executor.MakeArchiveDbTxProcessor(cfg), nil)
	if err == nil {
		t.Errorf("run must fail")
	}

	expectedErr := strings.TrimSpace("block: 2 transaction: 1\nintrinsic gas too low: have 0, want 53000")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

func TestVmAdb_ValidationDoesNotFailOnValidTransaction_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, true)
	cfg.Workers = 2
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			return consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		// we return correct expected data so tx does not fail
		archive.EXPECT().BeginTransaction(uint32(1)),
		archive.EXPECT().Exist(testingAddress).Return(true),
		archive.EXPECT().GetBalance(testingAddress).Return(new(uint256.Int).SetUint64(1)),
		archive.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		archive.EXPECT().GetCode(testingAddress).Return([]byte{}),

		archive.EXPECT().SetTxContext(gomock.Any(), 1),
		archive.EXPECT().Snapshot().Return(15),
		archive.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1000)),
		archive.EXPECT().SubBalance(gomock.Any(), gomock.Any(), gomock.Any()),
		archive.EXPECT().RevertToSnapshot(15),
		archive.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 1)), uint64(2), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
	)

	// run fails but not on validation
	err := run(cfg, provider, db, executor.MakeArchiveDbTxProcessor(cfg), nil)
	if err == nil {
		t.Errorf("run must fail")
	}

	expectedErr := strings.TrimSpace("block: 2 transaction: 1\nintrinsic gas too low: have 0, want 53000")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

func TestVmAdb_ValidationFailsOnInvalidTransaction_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, true)
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			return consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(1)),
		archive.EXPECT().Exist(testingAddress).Return(false), // address does not exist
		archive.EXPECT().GetBalance(testingAddress).Return(new(uint256.Int).SetUint64(1)),
		archive.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		archive.EXPECT().GetCode(testingAddress).Return([]byte{}),
	)

	err := run(cfg, provider, db, executor.MakeArchiveDbTxProcessor(cfg), nil)
	if err == nil {
		t.Errorf("validation must fail")
	}

	expectedErr := strings.TrimSpace("archive-db-validator err:\nblock 2 tx 1\n world-state input is not contained in the state-db\n   Account 0x0100000000000000000000000000000000000000 does not exist")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}

}

func TestVmAdb_ValidationFailsOnInvalidTransaction_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, true)
	cfg.Workers = 2
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			return consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(1)),
		archive.EXPECT().Exist(testingAddress).Return(false), // address does not exist
		archive.EXPECT().GetBalance(testingAddress).Return(new(uint256.Int).SetUint64(1)),
		archive.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		archive.EXPECT().GetCode(testingAddress).Return([]byte{}),
	)

	err := run(cfg, provider, db, executor.MakeArchiveDbTxProcessor(cfg), nil)
	if err == nil {
		t.Errorf("validation must fail")
	}

	expectedErr := strings.TrimSpace("archive-db-validator err:\nblock 2 tx 1\n world-state input is not contained in the state-db\n   Account 0x0100000000000000000000000000000000000000 does not exist")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}

}

// emptyTx is a dummy substate that will be processed without crashing.
var emptyTx = &substate.Substate{
	Env: &substate.Env{},
	Message: &substate.Message{
		GasPrice: big.NewInt(12),
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

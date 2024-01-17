package main

import (
	"math/big"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/transaction"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

var testingAddress = common.Address{1}

func TestVmAdb_AllDbEventsAreIssuedInOrder_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	db := state.NewMockStateDB(ctrl)
	archiveBlockOne := state.NewMockNonCommittableStateDB(ctrl)
	archiveBlockTwo := state.NewMockNonCommittableStateDB(ctrl)
	archiveBlockThree := state.NewMockNonCommittableStateDB(ctrl)

	cfg := &utils.Config{
		First:             2,
		Last:              4,
		ChainID:           utils.MainnetChainID,
		SkipPriming:       true,
		Workers:           1,
		ContinueOnFailure: true, // in this test we only check if blocks are being processed, any error can be ignored
		LogLevel:          "Critical",
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[transaction.SubstateData]) error {
			// Block 2
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewOldSubstateData(emptyTx)})
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 2, Data: transaction.NewOldSubstateData(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 3, Transaction: 1, Data: transaction.NewOldSubstateData(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 4, Transaction: utils.PseudoTx, Data: transaction.NewOldSubstateData(emptyTx)})
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
		archiveBlockOne.EXPECT().Prepare(gomock.Any(), 1),
		archiveBlockOne.EXPECT().Snapshot().Return(15),
		archiveBlockOne.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		archiveBlockOne.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		archiveBlockOne.EXPECT().RevertToSnapshot(15),
		archiveBlockOne.EXPECT().EndTransaction(),
		// Tx 2
		archiveBlockOne.EXPECT().BeginTransaction(uint32(2)),
		archiveBlockOne.EXPECT().Prepare(gomock.Any(), 2),
		archiveBlockOne.EXPECT().Snapshot().Return(16),
		archiveBlockOne.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		archiveBlockOne.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		archiveBlockOne.EXPECT().RevertToSnapshot(16),
		archiveBlockOne.EXPECT().EndTransaction(),
		archiveBlockOne.EXPECT().Release(),
		// Block 3
		db.EXPECT().GetArchiveState(uint64(2)).Return(archiveBlockTwo, nil),
		archiveBlockTwo.EXPECT().BeginTransaction(uint32(1)),
		archiveBlockTwo.EXPECT().Prepare(gomock.Any(), 1),
		archiveBlockTwo.EXPECT().Snapshot().Return(17),
		archiveBlockTwo.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		archiveBlockTwo.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		archiveBlockTwo.EXPECT().RevertToSnapshot(17),
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
	err := run(cfg, provider, db, executor.MakeArchiveDbProcessor(cfg), nil)
	if err != nil {
		if strings.Contains(err.Error(), "intrinsic gas too low") {
			return
		}
		t.Fatal("run failed")
	}
}

func TestVmAdb_AllDbEventsAreIssuedInOrder_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	db := state.NewMockStateDB(ctrl)
	archiveBlockOne := state.NewMockNonCommittableStateDB(ctrl)
	archiveBlockTwo := state.NewMockNonCommittableStateDB(ctrl)
	archiveBlockThree := state.NewMockNonCommittableStateDB(ctrl)

	cfg := &utils.Config{
		First:             2,
		Last:              4,
		ChainID:           utils.MainnetChainID,
		SkipPriming:       true,
		Workers:           4,
		ContinueOnFailure: true, // in this test we only check if blocks are being processed, any error can be ignored
		LogLevel:          "Critical",
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[transaction.SubstateData]) error {
			// Block 2
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewOldSubstateData(emptyTx)})
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 2, Data: transaction.NewOldSubstateData(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 3, Transaction: 1, Data: transaction.NewOldSubstateData(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 4, Transaction: utils.PseudoTx, Data: transaction.NewOldSubstateData(emptyTx)})
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
		archiveBlockOne.EXPECT().Prepare(gomock.Any(), 1),
		archiveBlockOne.EXPECT().Snapshot().Return(15),
		archiveBlockOne.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		archiveBlockOne.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		archiveBlockOne.EXPECT().RevertToSnapshot(15),
		archiveBlockOne.EXPECT().EndTransaction(),
		// Tx 2
		archiveBlockOne.EXPECT().BeginTransaction(uint32(2)),
		archiveBlockOne.EXPECT().Prepare(gomock.Any(), 2),
		archiveBlockOne.EXPECT().Snapshot().Return(19),
		archiveBlockOne.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		archiveBlockOne.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		archiveBlockOne.EXPECT().RevertToSnapshot(19),
		archiveBlockOne.EXPECT().EndTransaction(),

		archiveBlockOne.EXPECT().Release(),
	)
	// Block 3
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(2)).Return(archiveBlockTwo, nil),
		archiveBlockTwo.EXPECT().BeginTransaction(uint32(1)),
		archiveBlockTwo.EXPECT().Prepare(gomock.Any(), 1),
		archiveBlockTwo.EXPECT().Snapshot().Return(17),
		archiveBlockTwo.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		archiveBlockTwo.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		archiveBlockTwo.EXPECT().RevertToSnapshot(17),
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
	err := run(cfg, provider, db, executor.MakeArchiveDbProcessor(cfg), nil)
	if err != nil {
		if strings.Contains(err.Error(), "intrinsic gas too low") {
			return
		}
		t.Fatal("run failed")
	}
}

func TestVmAdb_AllTransactionsAreProcessedInOrder_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)
	ext := executor.NewMockExtension[transaction.SubstateData](ctrl)
	processor := executor.NewMockProcessor[transaction.SubstateData](ctrl)

	config := &utils.Config{
		First:    2,
		Last:     4,
		ChainID:  utils.MainnetChainID,
		LogLevel: "Critical",
		Workers:  1,
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[transaction.SubstateData]) error {
			// Block 2
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewOldSubstateData(emptyTx)})
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 2, Data: transaction.NewOldSubstateData(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 3, Transaction: 1, Data: transaction.NewOldSubstateData(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 4, Transaction: utils.PseudoTx, Data: transaction.NewOldSubstateData(emptyTx)})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running sequential mode with 1 worker,
	// all blocks and transactions need to be in order.

	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[transaction.SubstateData](2), gomock.Any()),

		// Block 2
		// Tx 1
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		ext.EXPECT().PreBlock(executor.AtBlock[transaction.SubstateData](2), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](2, 1), gomock.Any()),
		// Tx 2
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](2, 2), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](2, 2), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[transaction.SubstateData](2, 2), gomock.Any()),
		archive.EXPECT().Release(),

		// Block 3
		db.EXPECT().GetArchiveState(uint64(2)).Return(archive, nil),
		ext.EXPECT().PreBlock(executor.AtBlock[transaction.SubstateData](3), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](3, 1), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[transaction.SubstateData](3, 1), gomock.Any()),
		archive.EXPECT().Release(),

		// Block 4
		db.EXPECT().GetArchiveState(uint64(3)).Return(archive, nil),
		ext.EXPECT().PreBlock(executor.AtBlock[transaction.SubstateData](4), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](4, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[transaction.SubstateData](4, utils.PseudoTx), gomock.Any()),
		archive.EXPECT().Release(),

		ext.EXPECT().PostRun(executor.AtBlock[transaction.SubstateData](5), gomock.Any(), nil),
	)

	if err := run(config, provider, db, processor, []executor.Extension[transaction.SubstateData]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_AllTransactionsAreProcessed_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)
	ext := executor.NewMockExtension[transaction.SubstateData](ctrl)
	processor := executor.NewMockProcessor[transaction.SubstateData](ctrl)

	config := &utils.Config{
		First:    2,
		Last:     4,
		ChainID:  utils.MainnetChainID,
		LogLevel: "Critical",
		Workers:  4,
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[transaction.SubstateData]) error {
			// Block 2
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewOldSubstateData(emptyTx)})
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 2, Data: transaction.NewOldSubstateData(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 3, Transaction: 1, Data: transaction.NewOldSubstateData(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 4, Transaction: utils.PseudoTx, Data: transaction.NewOldSubstateData(emptyTx)})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running parallel mode with multiple workers block
	// order does not have to be preserved, only transaction order matters.
	ext.EXPECT().PreRun(executor.AtBlock[transaction.SubstateData](2), gomock.Any())

	// Block 2
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		ext.EXPECT().PreBlock(executor.AtBlock[transaction.SubstateData](2), gomock.Any()),
		// Tx 1
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](2, 1), gomock.Any()),
		// Tx 2
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](2, 2), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](2, 2), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtBlock[transaction.SubstateData](2), gomock.Any()),
		archive.EXPECT().Release(),
	)

	// Block 3
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(2)).Return(archive, nil),
		ext.EXPECT().PreBlock(executor.AtBlock[transaction.SubstateData](3), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](3, 1), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[transaction.SubstateData](3, 1), gomock.Any()),
		archive.EXPECT().Release(),
	)

	// Block 4
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(3)).Return(archive, nil),
		ext.EXPECT().PreBlock(executor.AtBlock[transaction.SubstateData](4), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](4, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[transaction.SubstateData](4, utils.PseudoTx), gomock.Any()),
		archive.EXPECT().Release(),
	)

	ext.EXPECT().PostRun(executor.AtBlock[transaction.SubstateData](5), gomock.Any(), nil)

	if err := run(config, provider, db, processor, []executor.Extension[transaction.SubstateData]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_ValidationDoesNotFailOnValidTransaction_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)

	cfg := &utils.Config{
		First:           2,
		Last:            4,
		ChainID:         utils.MainnetChainID,
		ValidateTxState: true,
		SkipPriming:     true,
		Workers:         1,
	}

	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[transaction.SubstateData]) error {
			return consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewOldSubstateData(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		// we return correct expected data so tx does not fail
		archive.EXPECT().Exist(testingAddress).Return(true),
		archive.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		archive.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		archive.EXPECT().GetCode(testingAddress).Return([]byte{}),

		archive.EXPECT().BeginTransaction(uint32(1)),
		archive.EXPECT().Prepare(gomock.Any(), 1),
		archive.EXPECT().Snapshot().Return(15),
		archive.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		archive.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		archive.EXPECT().RevertToSnapshot(15),
		archive.EXPECT().EndTransaction(),
	)

	// run fails but not on validation
	err := run(cfg, provider, db, executor.MakeArchiveDbProcessor(cfg), nil)
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
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)
	cfg := &utils.Config{
		First:           2,
		Last:            4,
		ChainID:         utils.MainnetChainID,
		ValidateTxState: true,
		SkipPriming:     true,
		Workers:         4,
	}

	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[transaction.SubstateData]) error {
			return consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewOldSubstateData(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		// we return correct expected data so tx does not fail
		archive.EXPECT().Exist(testingAddress).Return(true),
		archive.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		archive.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		archive.EXPECT().GetCode(testingAddress).Return([]byte{}),

		archive.EXPECT().BeginTransaction(uint32(1)),
		archive.EXPECT().Prepare(gomock.Any(), 1),
		archive.EXPECT().Snapshot().Return(15),
		archive.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		archive.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		archive.EXPECT().RevertToSnapshot(15),
		archive.EXPECT().EndTransaction(),
	)

	// run fails but not on validation
	err := run(cfg, provider, db, executor.MakeArchiveDbProcessor(cfg), nil)
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
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)
	cfg := &utils.Config{
		First:           2,
		Last:            4,
		ChainID:         utils.MainnetChainID,
		ValidateTxState: true,
		SkipPriming:     true,
	}

	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[transaction.SubstateData]) error {
			return consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewOldSubstateData(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		archive.EXPECT().Exist(testingAddress).Return(false), // address does not exist
		archive.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		archive.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		archive.EXPECT().GetCode(testingAddress).Return([]byte{}),
	)

	err := run(cfg, provider, db, executor.MakeArchiveDbProcessor(cfg), nil)
	if err == nil {
		t.Errorf("validation must fail")
	}

	expectedErr := strings.TrimSpace("archive-db-validator err:\nblock 2 tx 1\n input alloc is not contained in the state-db\n   Account 0x0100000000000000000000000000000000000000 does not exist")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}

}

func TestVmAdb_ValidationFailsOnInvalidTransaction_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)
	cfg := &utils.Config{
		First:           2,
		Last:            4,
		ChainID:         utils.MainnetChainID,
		ValidateTxState: true,
		SkipPriming:     true,
		Workers:         4,
	}

	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[transaction.SubstateData]) error {
			return consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewOldSubstateData(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		archive.EXPECT().Exist(testingAddress).Return(false), // address does not exist
		archive.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		archive.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		archive.EXPECT().GetCode(testingAddress).Return([]byte{}),
	)

	err := run(cfg, provider, db, executor.MakeArchiveDbProcessor(cfg), nil)
	if err == nil {
		t.Errorf("validation must fail")
	}

	expectedErr := strings.TrimSpace("archive-db-validator err:\nblock 2 tx 1\n input alloc is not contained in the state-db\n   Account 0x0100000000000000000000000000000000000000 does not exist")
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

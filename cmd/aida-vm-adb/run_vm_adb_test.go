package main

import (
	"math/big"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

var testingAddress = common.Address{1}

func TestVmAdb_AllDbEventsAreIssuedInOrder_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	db := state.NewMockStateDB(ctrl)
	archiveBlockOne := state.NewMockNonCommittableStateDB(ctrl)
	archiveBlockTwo := state.NewMockNonCommittableStateDB(ctrl)
	archiveBlockThree := state.NewMockNonCommittableStateDB(ctrl)
	cfg := &utils.Config{
		First:       2,
		Last:        4,
		ChainID:     utils.MainnetChainID,
		SkipPriming: true,
		Workers:     1,
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			// Block 2
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 1, Data: emptyTx})
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 2, Data: emptyTx})
			// Block 3
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 3, Transaction: 1, Data: emptyTx})
			// Block 4
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 4, Transaction: utils.PseudoTx, Data: emptyTx})
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
		// Block 3
		db.EXPECT().GetArchiveState(uint64(2)).Return(archiveBlockTwo, nil),
		archiveBlockTwo.EXPECT().BeginTransaction(uint32(1)),
		archiveBlockTwo.EXPECT().Prepare(gomock.Any(), 1),
		archiveBlockTwo.EXPECT().Snapshot().Return(17),
		archiveBlockTwo.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		archiveBlockTwo.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		archiveBlockTwo.EXPECT().RevertToSnapshot(17),
		archiveBlockTwo.EXPECT().EndTransaction(),
		// Block 4
		// Pseudo transaction do not use snapshots.
		db.EXPECT().GetArchiveState(uint64(3)).Return(archiveBlockThree, nil),
		archiveBlockThree.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		archiveBlockThree.EXPECT().EndTransaction(),
	)

	if err := run(cfg, provider, db, blockProcessor{cfg}, nil); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_AllDbEventsAreIssuedInOrder_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
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
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			// Block 2
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 1, Data: emptyTx})
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 2, Data: emptyTx})
			// Block 3
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 3, Transaction: 1, Data: emptyTx})
			// Block 4
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 4, Transaction: utils.PseudoTx, Data: emptyTx})
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
	)

	// Block 4
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(3)).Return(archiveBlockThree, nil),
		// Pseudo transaction do not use snapshots.
		archiveBlockThree.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		archiveBlockThree.EXPECT().EndTransaction(),
	)

	if err := run(cfg, provider, db, blockProcessor{cfg}, nil); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_AllTransactionsAreProcessedInOrder_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	db := state.NewMockStateDB(ctrl)
	ext := executor.NewMockExtension[*substate.Substate](ctrl)
	processor := executor.NewMockProcessor[*substate.Substate](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			// Block 2
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 1, Data: emptyTx})
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 2, Data: emptyTx})
			// Block 3
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 3, Transaction: 1, Data: emptyTx})
			// Block 4
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 4, Transaction: utils.PseudoTx, Data: emptyTx})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running sequential mode with 1 worker,
	// all blocks and transactions need to be in order.

	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[*substate.Substate](2), gomock.Any()),

		// Block 2
		// Tx 1
		db.EXPECT().GetArchiveState(uint64(1)),
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](2), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](2, 1), gomock.Any()),
		// Tx 2
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](2, 2), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](2, 2), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](2, 2), gomock.Any()),

		// Block 3
		db.EXPECT().GetArchiveState(uint64(2)),
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](3), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](3, 1), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](3, 1), gomock.Any()),

		// Block 4
		db.EXPECT().GetArchiveState(uint64(3)),
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](4), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](4, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](4, utils.PseudoTx), gomock.Any()),

		ext.EXPECT().PostRun(executor.AtBlock[*substate.Substate](5), gomock.Any(), nil),
	)

	if err := run(config, provider, db, processor, []executor.Extension[*substate.Substate]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_AllTransactionsAreProcessed_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	db := state.NewMockStateDB(ctrl)
	ext := executor.NewMockExtension[*substate.Substate](ctrl)
	processor := executor.NewMockProcessor[*substate.Substate](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			// Block 2
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 1, Data: emptyTx})
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 2, Data: emptyTx})
			// Block 3
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 3, Transaction: 1, Data: emptyTx})
			// Block 4
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 4, Transaction: utils.PseudoTx, Data: emptyTx})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running parallel mode with multiple workers block
	// order does not have to be preserved, only transaction order matters.
	ext.EXPECT().PreRun(executor.AtBlock[*substate.Substate](2), gomock.Any())

	// Block 2
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)),
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](2), gomock.Any()),
		// Tx 1
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](2, 1), gomock.Any()),
		// Tx 2
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](2, 2), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](2, 2), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtBlock[*substate.Substate](2), gomock.Any()),
	)

	// Block 3
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(2)),
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](3), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](3, 1), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](3, 1), gomock.Any()),
	)

	// Block 4
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(3)),
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](4), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](4, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](4, utils.PseudoTx), gomock.Any()),
	)

	ext.EXPECT().PostRun(executor.AtBlock[*substate.Substate](5), gomock.Any(), nil)

	if err := run(config, provider, db, processor, []executor.Extension[*substate.Substate]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_ValidationDoesNotFailOnValidTransaction_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			return consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 1, Data: testTx})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(1)),
		// we return correct expected data so tx does not fail
		archive.EXPECT().Exist(testingAddress).Return(true),
		archive.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		archive.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		archive.EXPECT().GetCode(testingAddress).Return([]byte{}),

		archive.EXPECT().Prepare(gomock.Any(), 1),
		archive.EXPECT().Snapshot().Return(15),
		archive.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		archive.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		archive.EXPECT().RevertToSnapshot(15),
		archive.EXPECT().EndTransaction(),
	)

	// run fails but not on validation
	err := run(cfg, provider, db, blockProcessor{cfg}, nil)
	if err == nil {
		t.Errorf("run must fail")
	}

	expectedErr := strings.TrimSpace("Block: 2 Transaction: 1\nintrinsic gas too low: have 0, want 53000")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

func TestVmAdb_ValidationDoesNotFailOnValidTransaction_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			return consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 1, Data: testTx})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(1)),
		// we return correct expected data so tx does not fail
		archive.EXPECT().Exist(testingAddress).Return(true),
		archive.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		archive.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		archive.EXPECT().GetCode(testingAddress).Return([]byte{}),

		archive.EXPECT().Prepare(gomock.Any(), 1),
		archive.EXPECT().Snapshot().Return(15),
		archive.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		archive.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		archive.EXPECT().RevertToSnapshot(15),
		archive.EXPECT().EndTransaction(),
	)

	// run fails but not on validation
	err := run(cfg, provider, db, blockProcessor{cfg}, nil)
	if err == nil {
		t.Errorf("run must fail")
	}

	expectedErr := strings.TrimSpace("Block: 2 Transaction: 1\nintrinsic gas too low: have 0, want 53000")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

func TestVmAdb_ValidationFailsOnInvalidTransaction_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			return consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 1, Data: testTx})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(1)),
		archive.EXPECT().Exist(testingAddress).Return(false), // address does not exist
		archive.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		archive.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		archive.EXPECT().GetCode(testingAddress).Return([]byte{}),
		archive.EXPECT().EndTransaction(),
	)

	err := run(cfg, provider, db, blockProcessor{cfg}, nil)
	if err == nil {
		t.Errorf("validation must fail")
	}

	expectedErr := strings.TrimSpace("Block: 2 Transaction: 1\nInput alloc is not contained in the stateDB.\n  " +
		"Account 0x0100000000000000000000000000000000000000 does not exist")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}

}

func TestVmAdb_ValidationFailsOnInvalidTransaction_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			return consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 1, Data: testTx})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(1)),
		archive.EXPECT().Exist(testingAddress).Return(false), // address does not exist
		archive.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		archive.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		archive.EXPECT().GetCode(testingAddress).Return([]byte{}),
		archive.EXPECT().EndTransaction(),
	)

	err := run(cfg, provider, db, blockProcessor{cfg}, nil)
	if err == nil {
		t.Errorf("validation must fail")
	}

	expectedErr := strings.TrimSpace("Block: 2 Transaction: 1\nInput alloc is not contained in the stateDB.\n  " +
		"Account 0x0100000000000000000000000000000000000000 does not exist")
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

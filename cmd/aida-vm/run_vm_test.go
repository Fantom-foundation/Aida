package main

import (
	"math/big"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/transaction"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substateCommon "github.com/Fantom-foundation/Substate/geth/common"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

var testingAddress = common.Address{1}

func TestVm_AllDbEventsAreIssuedInOrder_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	db := state.NewMockStateDB(ctrl)
	cfg := &utils.Config{
		First:             2,
		Last:              4,
		ChainID:           utils.MainnetChainID,
		SkipPriming:       true,
		Workers:           1,
		ContinueOnFailure: true, // in this test we only check if txs are being processed, any error can be ignored
		LogLevel:          "Critical",
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[transaction.SubstateData]) error {
			// Block 2
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewSubstateData(emptyTx)})
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 2, Data: transaction.NewSubstateData(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 3, Transaction: 1, Data: transaction.NewSubstateData(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 4, Transaction: utils.PseudoTx, Data: transaction.NewSubstateData(emptyTx)})
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
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().RevertToSnapshot(15),
		db.EXPECT().EndTransaction(),
		// Tx 2
		db.EXPECT().BeginTransaction(uint32(2)),
		db.EXPECT().Prepare(gomock.Any(), 2),
		db.EXPECT().Snapshot().Return(17),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().RevertToSnapshot(17),
		db.EXPECT().EndTransaction(),
		// Block 3
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(19),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().RevertToSnapshot(19),
		db.EXPECT().EndTransaction(),
		// Pseudo transaction do not use snapshots.
		db.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		db.EXPECT().EndTransaction(),
	)

	run(cfg, provider, db, executor.MakeLiveDbProcessor(cfg), nil)
}

func TestVm_AllDbEventsAreIssuedInOrder_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	db := state.NewMockStateDB(ctrl)
	cfg := &utils.Config{
		First:             2,
		Last:              4,
		ChainID:           utils.MainnetChainID,
		SkipPriming:       true,
		Workers:           4,
		ContinueOnFailure: true, // in this test we only check if txs are being processed, any error can be ignored
		LogLevel:          "Critical",
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[transaction.SubstateData]) error {
			// Block 2
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewSubstateData(emptyTx)})
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 2, Data: transaction.NewSubstateData(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 3, Transaction: 1, Data: transaction.NewSubstateData(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 4, Transaction: utils.PseudoTx, Data: transaction.NewSubstateData(emptyTx)})
			return nil
		})

	// The expectation is that all of those transactions are properly opened,
	// prepared, executed, and closed. Since we are running parallel mode with
	// multiple workers, tx order does not have to preserved.

	// Block 2 Tx 1
	gomock.InOrder(
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().RevertToSnapshot(15),
		db.EXPECT().EndTransaction(),
	)

	// Block 2 Tx 2
	gomock.InOrder(
		db.EXPECT().BeginTransaction(uint32(2)),
		db.EXPECT().Prepare(gomock.Any(), 2),
		db.EXPECT().Snapshot().Return(17),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().RevertToSnapshot(17),
		db.EXPECT().EndTransaction(),
	)

	// Block 3 Tx 1
	gomock.InOrder(
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(19),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().RevertToSnapshot(19),
		db.EXPECT().EndTransaction(),
	)

	// Block 4 Tx 1
	gomock.InOrder(
		// Pseudo transaction do not use snapshots.
		db.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		db.EXPECT().EndTransaction(),
	)

	run(cfg, provider, db, executor.MakeLiveDbProcessor(cfg), nil)
}

func TestVm_AllTransactionsAreProcessedInOrder_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	processor := executor.NewMockProcessor[transaction.SubstateData](ctrl)
	ext := executor.NewMockExtension[transaction.SubstateData](ctrl)
	cfg := &utils.Config{
		First:             2,
		Last:              4,
		ChainID:           utils.MainnetChainID,
		SkipPriming:       true,
		Workers:           1,
		ContinueOnFailure: true,
		LogLevel:          "Critical",
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[transaction.SubstateData]) error {
			// Block 2
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewSubstateData(emptyTx)})
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 2, Data: transaction.NewSubstateData(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 3, Transaction: 1, Data: transaction.NewSubstateData(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 4, Transaction: utils.PseudoTx, Data: transaction.NewSubstateData(emptyTx)})
			return nil
		})

	// The expectation is that all of those transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running sequential mode with 1 worker,
	// all blocks and transactions need to be in order.

	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[transaction.SubstateData](2), gomock.Any()),

		// Block 2
		// Tx 1
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](2, 1), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](2, 2), gomock.Any()),
		// Tx 2
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](2, 2), gomock.Any()),

		// Block 3
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](3, 1), gomock.Any()),

		// Block 4
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](4, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](4, utils.PseudoTx), gomock.Any()),

		ext.EXPECT().PostRun(executor.AtBlock[transaction.SubstateData](5), gomock.Any(), nil),
	)

	run(cfg, provider, nil, processor, []executor.Extension[transaction.SubstateData]{ext})
}

func TestVm_AllTransactionsAreProcessedInOrder_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	processor := executor.NewMockProcessor[transaction.SubstateData](ctrl)
	ext := executor.NewMockExtension[transaction.SubstateData](ctrl)
	cfg := &utils.Config{
		First:             2,
		Last:              4,
		ChainID:           utils.MainnetChainID,
		SkipPriming:       true,
		Workers:           4,
		ContinueOnFailure: true,
		LogLevel:          "Critical",
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[transaction.SubstateData]) error {
			// Block 2
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewSubstateData(emptyTx)})
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 2, Data: transaction.NewSubstateData(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 3, Transaction: 1, Data: transaction.NewSubstateData(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 4, Transaction: utils.PseudoTx, Data: transaction.NewSubstateData(emptyTx)})
			return nil
		})

	pre := ext.EXPECT().PreRun(executor.AtBlock[transaction.SubstateData](2), gomock.Any())
	post := ext.EXPECT().PostRun(executor.AtBlock[transaction.SubstateData](5), gomock.Any(), nil)

	// The expectation is that all of those transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running parallel mode with multiple workers tx
	// order does not have to be preserved.

	// Block 2
	// Tx 1
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](2, 1), gomock.Any()),
		post,
	)

	// Tx 2
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](2, 2), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](2, 2), gomock.Any()),
		post,
	)

	// Block 3
	// Tx 1
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](3, 1), gomock.Any()),
		post,
	)

	// Block 4
	// Tx 1
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[transaction.SubstateData](4, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[transaction.SubstateData](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[transaction.SubstateData](4, utils.PseudoTx), gomock.Any()),
		post,
	)

	if err := run(cfg, provider, nil, processor, []executor.Extension[transaction.SubstateData]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_ValidationDoesNotFailOnValidTransaction_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	db := state.NewMockStateDB(ctrl)
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
			return consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewSubstateData(testTx)})
		})

	gomock.InOrder(
		// we return correct expected data so tx does not fail
		db.EXPECT().Exist(testingAddress).Return(true),
		db.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		db.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		db.EXPECT().GetCode(testingAddress).Return([]byte{}),

		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(15),
		db.EXPECT().EndTransaction(),
	)

	// run fails but not on validation
	err := run(cfg, provider, db, executor.MakeLiveDbProcessor(cfg), nil)
	if err == nil {
		t.Errorf("run must fail")
	}

	// we expected error with low gas, which means the validation passed
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
			return consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewSubstateData(testTx)})
		})

	gomock.InOrder(
		// we return correct expected data so tx does not fail
		db.EXPECT().Exist(testingAddress).Return(true),
		db.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		db.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		db.EXPECT().GetCode(testingAddress).Return([]byte{}),

		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(15),
		db.EXPECT().EndTransaction(),
	)

	// run fails but not on validation
	err := run(cfg, provider, db, executor.MakeLiveDbProcessor(cfg), nil)
	if err == nil {
		t.Fatal("run must fail")
	}

	// we expected error with low gas, which means the validation passed
	expectedErr := strings.TrimSpace("block: 2 transaction: 1\nintrinsic gas too low: have 0, want 53000")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

func TestVm_ValidationFailsOnInvalidTransaction_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	db := state.NewMockStateDB(ctrl)
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
			return consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewSubstateData(testTx)})
		})

	gomock.InOrder(

		db.EXPECT().Exist(testingAddress).Return(false), // address does not exist
		db.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		db.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		db.EXPECT().GetCode(testingAddress).Return([]byte{}),
	)

	err := run(cfg, provider, db, executor.MakeLiveDbProcessor(cfg), nil)
	if err == nil {
		t.Errorf("validation must fail")
	}

	expectedErr := strings.TrimSpace("live-db-validator err:\nblock 2 tx 1\n input alloc is not contained in the state-db\n   Account 0x0100000000000000000000000000000000000000 does not exist")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

func TestVm_ValidationFailsOnInvalidTransaction_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[transaction.SubstateData](ctrl)
	db := state.NewMockStateDB(ctrl)
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
			return consumer(executor.TransactionInfo[transaction.SubstateData]{Block: 2, Transaction: 1, Data: transaction.NewSubstateData(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().Exist(testingAddress).Return(false), // address does not exist
		db.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		db.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		db.EXPECT().GetCode(testingAddress).Return([]byte{}),
	)

	err := run(cfg, provider, db, executor.MakeLiveDbProcessor(cfg), nil)
	if err == nil {
		t.Fatal("validation must fail")
	}

	expectedErr := strings.TrimSpace("live-db-validator err:\nblock 2 tx 1\n input alloc is not contained in the state-db\n   Account 0x0100000000000000000000000000000000000000 does not exist")
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
	},
	Result: &substate.Result{
		GasUsed: 1,
	},
}

// testTx is a dummy substate used for testing validation.
var testTx = &substate.Substate{
	InputAlloc: substate.Alloc{substateCommon.Address(testingAddress): substate.NewAccount(1, new(big.Int).SetUint64(1), []byte{})},
	Env:        &substate.Env{},
	Message: &substate.Message{
		GasPrice: big.NewInt(12),
	},
	Result: &substate.Result{
		GasUsed: 1,
	},
}

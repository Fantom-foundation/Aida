package main

import (
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

func TestVm_AllDbEventsAreIssuedInOrder_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.WithValidation](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.WithValidation]) error {
			// Block 2
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 2, Transaction: 2, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 3, Transaction: 1, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 4, Transaction: utils.PseudoTx, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
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
		// Pseudo txcontext do not use snapshots.
		db.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		db.EXPECT().EndTransaction(),
	)

	run(cfg, provider, db, executor.MakeLiveDbProcessor(cfg), nil)
}

func TestVm_AllDbEventsAreIssuedInOrder_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.WithValidation](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.WithValidation]) error {
			// Block 2
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 2, Transaction: 2, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 3, Transaction: 1, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 4, Transaction: utils.PseudoTx, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
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
		// Pseudo txcontext do not use snapshots.
		db.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		db.EXPECT().EndTransaction(),
	)

	run(cfg, provider, db, executor.MakeLiveDbProcessor(cfg), nil)
}

func TestVm_AllTransactionsAreProcessedInOrder_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.WithValidation](ctrl)
	processor := executor.NewMockProcessor[txcontext.WithValidation](ctrl)
	ext := executor.NewMockExtension[txcontext.WithValidation](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.WithValidation]) error {
			// Block 2
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 2, Transaction: 2, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 3, Transaction: 1, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 4, Transaction: utils.PseudoTx, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			return nil
		})

	// The expectation is that all of those transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running sequential mode with 1 worker,
	// all blocks and transactions need to be in order.

	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[txcontext.WithValidation](2), gomock.Any()),

		// Block 2
		// Tx 1
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.WithValidation](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.WithValidation](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.WithValidation](2, 1), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.WithValidation](2, 2), gomock.Any()),
		// Tx 2
		processor.EXPECT().Process(executor.AtTransaction[txcontext.WithValidation](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.WithValidation](2, 2), gomock.Any()),

		// Block 3
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.WithValidation](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.WithValidation](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.WithValidation](3, 1), gomock.Any()),

		// Block 4
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.WithValidation](4, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.WithValidation](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.WithValidation](4, utils.PseudoTx), gomock.Any()),

		ext.EXPECT().PostRun(executor.AtBlock[txcontext.WithValidation](5), gomock.Any(), nil),
	)

	run(cfg, provider, nil, processor, []executor.Extension[txcontext.WithValidation]{ext})
}

func TestVm_AllTransactionsAreProcessedInOrder_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.WithValidation](ctrl)
	processor := executor.NewMockProcessor[txcontext.WithValidation](ctrl)
	ext := executor.NewMockExtension[txcontext.WithValidation](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.WithValidation]) error {
			// Block 2
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 2, Transaction: 2, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 3, Transaction: 1, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 4, Transaction: utils.PseudoTx, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			return nil
		})

	pre := ext.EXPECT().PreRun(executor.AtBlock[txcontext.WithValidation](2), gomock.Any())
	post := ext.EXPECT().PostRun(executor.AtBlock[txcontext.WithValidation](5), gomock.Any(), nil)

	// The expectation is that all of those transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running parallel mode with multiple workers tx
	// order does not have to be preserved.

	// Block 2
	// Tx 1
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.WithValidation](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.WithValidation](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.WithValidation](2, 1), gomock.Any()),
		post,
	)

	// Tx 2
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.WithValidation](2, 2), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.WithValidation](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.WithValidation](2, 2), gomock.Any()),
		post,
	)

	// Block 3
	// Tx 1
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.WithValidation](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.WithValidation](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.WithValidation](3, 1), gomock.Any()),
		post,
	)

	// Block 4
	// Tx 1
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.WithValidation](4, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.WithValidation](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.WithValidation](4, utils.PseudoTx), gomock.Any()),
		post,
	)

	if err := run(cfg, provider, nil, processor, []executor.Extension[txcontext.WithValidation]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_ValidationDoesNotFailOnValidTransaction_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.WithValidation](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.WithValidation]) error {
			return consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContextWithValidation(testTx)})
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
	expectedErr := strings.TrimSpace("block: 2 txcontext: 1\nintrinsic gas too low: have 0, want 53000")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

func TestVmAdb_ValidationDoesNotFailOnValidTransaction_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.WithValidation](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.WithValidation]) error {
			return consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContextWithValidation(testTx)})
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
	expectedErr := strings.TrimSpace("block: 2 txcontext: 1\nintrinsic gas too low: have 0, want 53000")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

func TestVm_ValidationFailsOnInvalidTransaction_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.WithValidation](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.WithValidation]) error {
			return consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContextWithValidation(testTx)})
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
	provider := executor.NewMockProvider[txcontext.WithValidation](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.WithValidation]) error {
			return consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContextWithValidation(testTx)})
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

package main

import (
	"errors"
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

func TestVmSdb_Substate_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[executor.TransactionData](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[executor.TransactionData]) error {
			// Block 2
			consumer(executor.TransactionInfo[executor.TransactionData]{Block: 2, Transaction: 1, Data: executor.NewSubstateData(emptyTx)})
			consumer(executor.TransactionInfo[executor.TransactionData]{Block: 2, Transaction: 2, Data: executor.NewSubstateData(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[executor.TransactionData]{Block: 3, Transaction: 1, Data: executor.NewSubstateData(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[executor.TransactionData]{Block: 4, Transaction: utils.PseudoTx, Data: executor.NewSubstateData(emptyTx)})
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
		db.EXPECT().EndTransaction(),
		// Tx 2
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		db.EXPECT().BeginTransaction(uint32(2)),
		db.EXPECT().Prepare(gomock.Any(), 2),
		db.EXPECT().Snapshot().Return(17),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(17),
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
	err := runSubstates(cfg, provider, db, executor.MakeLiveDbProcessor(cfg), nil)
	if err != nil {
		errors.Unwrap(err)
		if strings.Contains(err.Error(), "intrinsic gas too low") {
			return
		}
		t.Fatal("run failed")
	}
}

func TestVmSdb_Substate_AllTransactionsAreProcessedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[executor.TransactionData](ctrl)
	db := state.NewMockStateDB(ctrl)
	ext := executor.NewMockExtension[executor.TransactionData](ctrl)
	processor := executor.NewMockProcessor[executor.TransactionData](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[executor.TransactionData]) error {
			// Block 2
			consumer(executor.TransactionInfo[executor.TransactionData]{Block: 2, Transaction: 1, Data: executor.NewSubstateData(emptyTx)})
			consumer(executor.TransactionInfo[executor.TransactionData]{Block: 2, Transaction: 2, Data: executor.NewSubstateData(emptyTx)})
			// Block 3
			consumer(executor.TransactionInfo[executor.TransactionData]{Block: 3, Transaction: 1, Data: executor.NewSubstateData(emptyTx)})
			// Block 4
			consumer(executor.TransactionInfo[executor.TransactionData]{Block: 4, Transaction: utils.PseudoTx, Data: executor.NewSubstateData(emptyTx)})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running sequential mode with 1 worker,
	// all block and transactions need to be in order.
	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[executor.TransactionData](2), gomock.Any()),

		// Block 2
		// Tx 1
		ext.EXPECT().PreBlock(executor.AtBlock[executor.TransactionData](2), gomock.Any()),
		db.EXPECT().BeginBlock(uint64(2)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[executor.TransactionData](2, 1), gomock.Any()),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		processor.EXPECT().Process(executor.AtTransaction[executor.TransactionData](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[executor.TransactionData](2, 1), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[executor.TransactionData](2, 2), gomock.Any()),
		// Tx 2
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		processor.EXPECT().Process(executor.AtTransaction[executor.TransactionData](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[executor.TransactionData](2, 2), gomock.Any()),
		db.EXPECT().EndBlock(),
		ext.EXPECT().PostBlock(executor.AtTransaction[executor.TransactionData](2, 2), gomock.Any()),

		// Block 3
		ext.EXPECT().PreBlock(executor.AtBlock[executor.TransactionData](3), gomock.Any()),
		db.EXPECT().BeginBlock(uint64(3)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[executor.TransactionData](3, 1), gomock.Any()),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(3)),
		processor.EXPECT().Process(executor.AtTransaction[executor.TransactionData](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[executor.TransactionData](3, 1), gomock.Any()),
		db.EXPECT().EndBlock(),
		ext.EXPECT().PostBlock(executor.AtTransaction[executor.TransactionData](3, 1), gomock.Any()),

		// Block 4
		ext.EXPECT().PreBlock(executor.AtBlock[executor.TransactionData](4), gomock.Any()),
		db.EXPECT().BeginBlock(uint64(4)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[executor.TransactionData](4, utils.PseudoTx), gomock.Any()),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(4)),
		processor.EXPECT().Process(executor.AtTransaction[executor.TransactionData](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[executor.TransactionData](4, utils.PseudoTx), gomock.Any()),
		db.EXPECT().EndBlock(),
		ext.EXPECT().PostBlock(executor.AtTransaction[executor.TransactionData](4, utils.PseudoTx), gomock.Any()),

		ext.EXPECT().PostRun(executor.AtBlock[executor.TransactionData](5), gomock.Any(), nil),
	)

	if err := runSubstates(cfg, provider, db, processor, []executor.Extension[executor.TransactionData]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmSdb_Substate_ValidationDoesNotFailOnValidTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[executor.TransactionData](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[executor.TransactionData]) error {
			return consumer(executor.TransactionInfo[executor.TransactionData]{Block: 2, Transaction: 1, Data: executor.NewSubstateData(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().BeginBlock(uint64(2)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),

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
	err := runSubstates(cfg, provider, db, executor.MakeLiveDbProcessor(cfg), nil)
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
	provider := executor.NewMockProvider[executor.TransactionData](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[executor.TransactionData]) error {
			return consumer(executor.TransactionInfo[executor.TransactionData]{Block: 2, Transaction: 1, Data: executor.NewSubstateData(testTx)})
		})

	gomock.InOrder(
		db.EXPECT().BeginBlock(uint64(2)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		db.EXPECT().Exist(testingAddress).Return(false), // address does not exist
		db.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		db.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		db.EXPECT().GetCode(testingAddress).Return([]byte{}),
	)

	err := runSubstates(cfg, provider, db, executor.MakeLiveDbProcessor(cfg), nil)
	if err == nil {
		t.Errorf("validation must fail")
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

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

func TestVmSdb_Substate_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	db := state.NewMockStateDB(ctrl)
	cfg := &utils.Config{
		First:       2,
		Last:        4,
		ChainID:     utils.MainnetChainID,
		SkipPriming: true,
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

	if err := run(cfg, provider, db, true, txProcessor{cfg}, nil); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmSdb_Substate_AllTransactionsAreProcessedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	db := state.NewMockStateDB(ctrl)
	ext := executor.NewMockExtension[*substate.Substate](ctrl)
	processor := executor.NewMockProcessor[*substate.Substate](ctrl)
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
	// all block and transactions need to be in order.
	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[*substate.Substate](2), gomock.Any()),

		// Block 2
		// Tx 1
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](2), gomock.Any()),
		db.EXPECT().BeginBlock(uint64(2)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](2, 1), gomock.Any()),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](2, 1), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](2, 2), gomock.Any()),
		// Tx 2
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](2, 2), gomock.Any()),
		db.EXPECT().EndBlock(),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](2, 2), gomock.Any()),

		// Block 3
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](3), gomock.Any()),
		db.EXPECT().BeginBlock(uint64(3)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](3, 1), gomock.Any()),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(3)),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](3, 1), gomock.Any()),
		db.EXPECT().EndBlock(),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](3, 1), gomock.Any()),

		// Block 4
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](4), gomock.Any()),
		db.EXPECT().BeginBlock(uint64(4)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](4, utils.PseudoTx), gomock.Any()),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(4)),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](4, utils.PseudoTx), gomock.Any()),
		db.EXPECT().EndBlock(),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](4, utils.PseudoTx), gomock.Any()),

		ext.EXPECT().PostRun(executor.AtBlock[*substate.Substate](5), gomock.Any(), nil),
	)

	if err := run(cfg, provider, db, true, processor, []executor.Extension[*substate.Substate]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmSdb_Substate_ValidationDoesNotFailOnValidTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			return consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 1, Data: testTx})
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
		db.EXPECT().EndTransaction(),
	)

	// run fails but not on validation
	err := run(cfg, provider, db, true, txProcessor{cfg}, nil)
	if err == nil {
		t.Errorf("run must fail")
	}

	expectedErr := strings.TrimSpace("Block: 2 Transaction: 1\nintrinsic gas too low: have 0, want 53000")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}
}

func TestVmSdb_Substate_ValidationFailsOnInvalidTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
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
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			return consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 1, Data: testTx})
		})

	gomock.InOrder(
		db.EXPECT().BeginBlock(uint64(2)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Exist(testingAddress).Return(false), // address does not exist
		db.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		db.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		db.EXPECT().GetCode(testingAddress).Return([]byte{}),
		db.EXPECT().EndTransaction(),
	)

	err := run(cfg, provider, db, true, txProcessor{cfg}, nil)
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

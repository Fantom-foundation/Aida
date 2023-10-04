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

func TestVmSdb_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	db := state.NewMockStateDB(ctrl)
	config := &utils.Config{
		First:       0,
		Last:        2,
		ChainID:     utils.MainnetChainID,
		SkipPriming: true,
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(0, 3, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			// block 0
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 0, Transaction: 1, Payload: emptyTx})
			// block 2
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: 3, Payload: emptyTx})
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 2, Transaction: utils.PseudoTx, Payload: emptyTx})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	gomock.InOrder(
		// Block 0
		db.EXPECT().BeginBlock(uint64(0)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(0)),
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(15),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),
		// Begin Block 2
		db.EXPECT().BeginBlock(uint64(2)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		db.EXPECT().BeginTransaction(uint32(3)),
		db.EXPECT().Prepare(gomock.Any(), 3),
		db.EXPECT().Snapshot().Return(17),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(17),
		db.EXPECT().EndTransaction(),
		// Pseudo transaction do not use snapshots.
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
		db.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),
	)

	if err := run(config, provider, db, true); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmSdb_ValidationDoesNotFailOnValidTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	db := state.NewMockStateDB(ctrl)
	config := &utils.Config{
		First:           0,
		Last:            2,
		ChainID:         utils.MainnetChainID,
		ValidateTxState: true,
		SkipPriming:     true,
	}

	provider.EXPECT().
		Run(0, 3, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			return consumer(executor.TransactionInfo[*substate.Substate]{Block: 0, Transaction: 1, Payload: testTx})
		})

	gomock.InOrder(
		db.EXPECT().BeginBlock(uint64(0)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(0)),
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
	err := run(config, provider, db, true)
	if err == nil {
		t.Errorf("run must fail")
	}

	expectedErr := strings.TrimSpace("Block: 0 Transaction: 1\nintrinsic gas too low: have 0, want 53000")
	returnedErr := strings.TrimSpace(err.Error())

	if strings.Compare(returnedErr, expectedErr) != 0 {
		t.Errorf("unexpected error; \n got: %v\n want: %v", err.Error(), expectedErr)
	}

}

func TestVmSdb_ValidationFailsOnInvalidTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	db := state.NewMockStateDB(ctrl)
	config := &utils.Config{
		First:           0,
		Last:            2,
		ChainID:         utils.MainnetChainID,
		ValidateTxState: true,
		SkipPriming:     true,
	}

	provider.EXPECT().
		Run(0, 3, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			return consumer(executor.TransactionInfo[*substate.Substate]{Block: 0, Transaction: 1, Payload: testTx})
		})

	gomock.InOrder(
		db.EXPECT().BeginBlock(uint64(0)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(0)),
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Exist(testingAddress).Return(false), // address does not exist
		db.EXPECT().GetBalance(testingAddress).Return(new(big.Int).SetUint64(1)),
		db.EXPECT().GetNonce(testingAddress).Return(uint64(1)),
		db.EXPECT().GetCode(testingAddress).Return([]byte{}),
		db.EXPECT().EndTransaction(),
	)

	err := run(config, provider, db, true)
	if err == nil {
		t.Errorf("validation must fail")
	}

	expectedErr := strings.TrimSpace("Block: 0 Transaction: 1\nInput alloc is not contained in the stateDB.\n  " +
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

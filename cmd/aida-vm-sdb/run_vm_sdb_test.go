package main

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"go.uber.org/mock/gomock"
)

func TestVmSdb_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := executor.NewMockSubstateProvider(ctrl)
	db := state.NewMockStateDB(ctrl)
	config := &utils.Config{
		First:   0,
		Last:    2,
		ChainID: utils.MainnetChainID,
	}

	// Simulate the execution of three transactions in two blocks.
	substate.EXPECT().
		Run(0, 3, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer) error {
			// block 0
			consumer(executor.TransactionInfo{Block: 0, Transaction: 1, Substate: emptyTx}, nil)
			// block 2
			consumer(executor.TransactionInfo{Block: 2, Transaction: 3, Substate: emptyTx}, nil)
			consumer(executor.TransactionInfo{Block: 2, Transaction: utils.PseudoTx, Substate: emptyTx}, nil)
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

	if err := run(config, substate, db); err != nil {
		t.Errorf("run failed: %v", err)
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

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

func TestVmAdb_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := executor.NewMockSubstateProvider(ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockStateDB(ctrl)
	config := &utils.Config{
		First:    1,
		Last:     1,
		ChainID:  utils.MainnetChainID,
		LogLevel: "Critical",
	}

	// Simulate the execution of three transactions in two blocks.
	substate.EXPECT().
		Run(1, 2, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer) error {
			consumer(executor.TransactionInfo{Block: 1, Transaction: 1, Substate: emptyTx})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	gomock.InOrder(
		db.EXPECT().BeginBlock(uint64(1)),
		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(1)),
		db.EXPECT().GetArchiveState(uint64(0)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(1)),
		archive.EXPECT().Prepare(gomock.Any(), 1),
		archive.EXPECT().Snapshot().Return(15),
		archive.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		archive.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		archive.EXPECT().RevertToSnapshot(15),
		archive.EXPECT().EndTransaction(),
	)

	if err := run(config, substate, db, true); err != nil {
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

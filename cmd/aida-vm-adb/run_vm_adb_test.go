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
	archive := state.NewMockNonCommittableStateDB(ctrl)
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
			consumer(executor.TransactionInfo{Block: 1, Transaction: 0, Substate: emptyTx})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(0)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(0)),
		archive.EXPECT().Prepare(gomock.Any(), 0),
		archive.EXPECT().Snapshot().Return(15),
		archive.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		archive.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		archive.EXPECT().RevertToSnapshot(15),
		archive.EXPECT().EndTransaction(),
	)

	if err := run(config, substate, db, blockProcessor{config}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_AllDbEventsAreIssuedInOrderMultipleBlocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := executor.NewMockSubstateProvider(ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)
	config := &utils.Config{
		First:    1,
		Last:     2,
		ChainID:  utils.MainnetChainID,
		LogLevel: "Critical",
	}

	// Simulate the execution of three transactions in two blocks.
	substate.EXPECT().
		Run(1, 3, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer) error {
			consumer(executor.TransactionInfo{Block: 1, Transaction: 0, Substate: emptyTx})
			consumer(executor.TransactionInfo{Block: 2, Transaction: 0, Substate: emptyTx})
			return nil
		})

		// The expectation is that all of those blocks and transactions
		// are properly opened, prepared, executed, and closed.
		// gomock.InOrder(
	db.EXPECT().GetArchiveState(uint64(0)).Return(archive, nil)
	archive.EXPECT().BeginTransaction(uint32(0))
	archive.EXPECT().Prepare(gomock.Any(), 0)
	archive.EXPECT().Snapshot().Return(15)
	archive.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000))
	archive.EXPECT().SubBalance(gomock.Any(), gomock.Any())
	archive.EXPECT().RevertToSnapshot(15)
	archive.EXPECT().EndTransaction()
	// )

	db.EXPECT().GetArchiveState(uint64(1)).Return(archive, nil)
	archive.EXPECT().BeginTransaction(uint32(0))
	archive.EXPECT().Prepare(gomock.Any(), 0)
	archive.EXPECT().Snapshot().Return(15)
	archive.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000))
	archive.EXPECT().SubBalance(gomock.Any(), gomock.Any())
	archive.EXPECT().RevertToSnapshot(15)
	archive.EXPECT().EndTransaction()

	if err := run(config, substate, db, blockProcessor{config}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_AllDbEventsAreIssuedInOrderMultipleBlocksMultipleWorkers(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := executor.NewMockSubstateProvider(ctrl)
	db := state.NewMockStateDB(ctrl)
	// archive := state.NewMockNonCommittableStateDB(ctrl)
	processor := executor.NewMockProcessor(ctrl)
	config := &utils.Config{
		First:    10,
		Last:     13,
		ChainID:  utils.MainnetChainID,
		LogLevel: "Critical",
		Workers:  1,
	}

	// Simulate the execution of three transactions in two blocks.
	substate.EXPECT().
		Run(10, 14, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer) error {
			consumer(executor.TransactionInfo{Block: 10, Transaction: 3, Substate: emptyTx})
			consumer(executor.TransactionInfo{Block: 11, Transaction: 5, Substate: emptyTx})
			consumer(executor.TransactionInfo{Block: 12, Transaction: 7, Substate: emptyTx})
			consumer(executor.TransactionInfo{Block: 13, Transaction: 9, Substate: emptyTx})
			return nil
		})

	processor.EXPECT().Process(executor.AtTransaction(10, 3), gomock.Any())
	db.EXPECT().GetArchiveState(uint64(9))
	processor.EXPECT().Process(executor.AtTransaction(11, 5), gomock.Any())
	db.EXPECT().GetArchiveState(uint64(10))
	processor.EXPECT().Process(executor.AtTransaction(12, 7), gomock.Any())
	db.EXPECT().GetArchiveState(uint64(11))
	processor.EXPECT().Process(executor.AtTransaction(13, 9), gomock.Any())
	db.EXPECT().GetArchiveState(uint64(12))

	if err := run(config, substate, db, processor); err != nil {
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

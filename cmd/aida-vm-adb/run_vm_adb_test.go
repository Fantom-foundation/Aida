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
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)
	config := &utils.Config{
		First:    1,
		Last:     1,
		ChainID:  utils.MainnetChainID,
		LogLevel: "Critical",
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(1, 2, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 1, Transaction: 1, Data: emptyTx})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(0)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(1)),
		archive.EXPECT().Prepare(gomock.Any(), 1),
		archive.EXPECT().Snapshot().Return(15),
		archive.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		archive.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		archive.EXPECT().RevertToSnapshot(15),
		archive.EXPECT().EndTransaction(),
	)

	if err := run(config, provider, db, blockProcessor{config}, nil); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_AllBlocksAreProcessedInOrderInSequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	db := state.NewMockStateDB(ctrl)
	ext := executor.NewMockExtension[*substate.Substate](ctrl)
	processor := executor.NewMockProcessor[*substate.Substate](ctrl)
	config := &utils.Config{
		First:    10,
		Last:     13,
		ChainID:  utils.MainnetChainID,
		LogLevel: "Critical",
		Workers:  1,
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(10, 14, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 10, Transaction: 3, Data: emptyTx})
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 11, Transaction: 5, Data: emptyTx})
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 12, Transaction: 7, Data: emptyTx})
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 13, Transaction: 9, Data: emptyTx})
			return nil
		})

	// order of the blocks needs to be preserved in sequential mode
	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[*substate.Substate](10), gomock.Any()),

		db.EXPECT().GetArchiveState(uint64(9)),
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](10), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](10, 3), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](10, 3), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](10, 3), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](10, 3), gomock.Any()),

		db.EXPECT().GetArchiveState(uint64(10)),
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](11), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](11, 5), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](11, 5), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](11, 5), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](11, 5), gomock.Any()),

		db.EXPECT().GetArchiveState(uint64(11)),
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](12), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](12, 7), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](12, 7), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](12, 7), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](12, 7), gomock.Any()),

		db.EXPECT().GetArchiveState(uint64(12)),
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](13), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](13, 9), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](13, 9), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](13, 9), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](13, 9), gomock.Any()),

		ext.EXPECT().PostRun(executor.AtBlock[*substate.Substate](14), gomock.Any(), nil),
	)

	if err := run(config, provider, db, processor, []executor.Extension[*substate.Substate]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestVmAdb_AllBlocksAreProcessedInParallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	db := state.NewMockStateDB(ctrl)
	ext := executor.NewMockExtension[*substate.Substate](ctrl)
	processor := executor.NewMockProcessor[*substate.Substate](ctrl)
	config := &utils.Config{
		First:    10,
		Last:     13,
		ChainID:  utils.MainnetChainID,
		LogLevel: "Critical",
		Workers:  4,
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(10, 14, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*substate.Substate]) error {
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 10, Transaction: 3, Data: emptyTx})
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 11, Transaction: 5, Data: emptyTx})
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 12, Transaction: 7, Data: emptyTx})
			consumer(executor.TransactionInfo[*substate.Substate]{Block: 13, Transaction: 9, Data: emptyTx})
			return nil
		})

	// we cannot guarantee order of the blocks in parallel mode
	// though each call in parallel need to preserve order

	ext.EXPECT().PreRun(executor.AtBlock[*substate.Substate](10), gomock.Any())

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(9)),
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](10), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](10, 3), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](10, 3), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](10, 3), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](10, 3), gomock.Any()),
	)

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(10)),
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](11), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](11, 5), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](11, 5), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](11, 5), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](11, 5), gomock.Any()),
	)

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(11)),
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](12), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](12, 7), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](12, 7), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](12, 7), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](12, 7), gomock.Any()),
	)

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(12)),
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](13), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](13, 9), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](13, 9), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](13, 9), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[*substate.Substate](13, 9), gomock.Any()),
	)

	ext.EXPECT().PostRun(executor.AtBlock[*substate.Substate](14), gomock.Any(), nil)

	if err := run(config, provider, db, processor, []executor.Extension[*substate.Substate]{ext}); err != nil {
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

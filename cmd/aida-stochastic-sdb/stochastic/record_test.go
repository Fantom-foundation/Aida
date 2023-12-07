package stochastic

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

var testingAddress = common.Address{1}

func TestStochastic_Record_AllTransactionsAreProcessedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	processor := executor.NewMockProcessor[*substate.Substate](ctrl)
	ext := executor.NewMockExtension[*substate.Substate](ctrl)
	path := t.TempDir() + "test_file"
	cfg := &utils.Config{
		First:            10,
		Last:             11,
		ChainID:          utils.MainnetChainID,
		SkipPriming:      true,
		SyncPeriodLength: 1,
		Output:           path,
	}

	provider.EXPECT().
		Run(10, 12, gomock.Any()).
		DoAndReturn(func(from int, to int, consumer executor.Consumer[*substate.Substate]) error {
			for i := from; i < to; i++ {
				consumer(executor.TransactionInfo[*substate.Substate]{Block: i, Transaction: 3, Data: emptyTx})
				consumer(executor.TransactionInfo[*substate.Substate]{Block: i, Transaction: utils.PseudoTx, Data: emptyTx})
			}
			return nil
		})

	// All transactions are processed in order
	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[*substate.Substate](10), gomock.Any()),

		// block 10
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](10), gomock.Any()),

		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](10, 3), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](10, 3), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](10, 3), gomock.Any()),

		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](10, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](10, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](10, utils.PseudoTx), gomock.Any()),

		ext.EXPECT().PostBlock(executor.AtBlock[*substate.Substate](10), gomock.Any()),

		// block 11
		ext.EXPECT().PreBlock(executor.AtBlock[*substate.Substate](11), gomock.Any()),

		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](11, 3), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](11, 3), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](11, 3), gomock.Any()),

		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](11, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](11, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](11, utils.PseudoTx), gomock.Any()),

		ext.EXPECT().PostBlock(executor.AtBlock[*substate.Substate](11), gomock.Any()),

		ext.EXPECT().PostRun(executor.AtBlock[*substate.Substate](12), gomock.Any(), nil),
	)

	if err := record(cfg, provider, nil, processor, []executor.Extension[*substate.Substate]{ext}); err != nil {
		t.Errorf("record failed: %v", err)
	}
}

func TestStochastic_Record_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	db := state.NewMockStateDB(ctrl)
	path := t.TempDir() + "test_file"
	cfg := &utils.Config{
		First:            2,
		Last:             4,
		ChainID:          utils.MainnetChainID,
		SkipPriming:      true,
		SyncPeriodLength: 1,
		Output:           path,
		Workers:          1,
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
		// Tx 1
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(15),
		db.EXPECT().EndTransaction(),
		// Tx 2
		db.EXPECT().BeginTransaction(uint32(2)),
		db.EXPECT().Prepare(gomock.Any(), 2),
		db.EXPECT().Snapshot().Return(17),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(17),
		db.EXPECT().EndTransaction(),
		// Block 3
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(19),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(1000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().RevertToSnapshot(19),
		db.EXPECT().EndTransaction(),
		// Pseudo transaction do not use snapshots.
		db.EXPECT().BeginTransaction(uint32(utils.PseudoTx)),
		db.EXPECT().EndTransaction(),
	)

	if err := record(cfg, provider, db, executor.MakeLiveDbProcessor(cfg), nil); err != nil {
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

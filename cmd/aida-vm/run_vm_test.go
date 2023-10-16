package main

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"go.uber.org/mock/gomock"
)

func TestVmSdb_TransactionsAreExecutedForCorrectRange(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	processor := executor.NewMockProcessor[*substate.Substate](ctrl)
	ext := executor.NewMockExtension[*substate.Substate](ctrl)

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(10, 12, gomock.Any()).
		DoAndReturn(func(from int, to int, consumer executor.Consumer[*substate.Substate]) error {
			for i := from; i < to; i++ {
				consumer(executor.TransactionInfo[*substate.Substate]{Block: i, Transaction: 3, Data: emptyTx})
				consumer(executor.TransactionInfo[*substate.Substate]{Block: i, Transaction: utils.PseudoTx, Data: emptyTx})
			}
			return nil
		})

	pre := ext.EXPECT().PreRun(executor.AtBlock[*substate.Substate](10), gomock.Any())
	post := ext.EXPECT().PostRun(executor.AtBlock[*substate.Substate](12), gomock.Any(), nil)

	// All transactions are processed, but in no specific order.
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](10, 3), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](10, 3), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](10, 3), gomock.Any()),
		post,
	)
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](10, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](10, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](10, utils.PseudoTx), gomock.Any()),
		post,
	)
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](11, 3), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](11, 3), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](11, 3), gomock.Any()),
		post,
	)
	gomock.InOrder(
		pre,
		ext.EXPECT().PreTransaction(executor.AtTransaction[*substate.Substate](11, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[*substate.Substate](11, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[*substate.Substate](11, utils.PseudoTx), gomock.Any()),
		post,
	)

	config := &utils.Config{}
	config.ChainID = 250
	config.Workers = 4
	config.First = 10
	config.Last = 11
	if err := run(config, provider, processor, []executor.Extension[*substate.Substate]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

// emptyTx is a dummy substate that will be processed without crashing.
var emptyTx = &substate.Substate{
	Env: &substate.SubstateEnv{},
	Message: &substate.SubstateMessage{
		Gas:      10000,
		GasPrice: big.NewInt(0),
	},
	Result: &substate.SubstateResult{
		GasUsed: 1,
	},
}

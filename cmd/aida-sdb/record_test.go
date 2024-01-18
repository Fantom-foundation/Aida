package main

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/txcontext"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

var testingAddress = common.Address{1}

func TestSdbRecord_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.WithValidation](ctrl)
	processor := executor.NewMockProcessor[txcontext.WithValidation](ctrl)
	ext := executor.NewMockExtension[txcontext.WithValidation](ctrl)
	path := t.TempDir() + "test_trace"
	cfg := &utils.Config{
		First:            10,
		Last:             11,
		ChainID:          utils.MainnetChainID,
		SkipPriming:      true,
		TraceFile:        path,
		SyncPeriodLength: 1,
	}

	provider.EXPECT().
		Run(10, 12, gomock.Any()).
		DoAndReturn(func(from int, to int, consumer executor.Consumer[txcontext.WithValidation]) error {
			for i := from; i < to; i++ {
				consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: i, Transaction: 3, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
				consumer(executor.TransactionInfo[txcontext.WithValidation]{Block: i, Transaction: utils.PseudoTx, Data: substatecontext.NewTxContextWithValidation(emptyTx)})
			}
			return nil
		})

	// All transactions are processed in order
	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[txcontext.WithValidation](10), gomock.Any()),

		// block 10
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.WithValidation](10, 3), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.WithValidation](10, 3), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.WithValidation](10, 3), gomock.Any()),

		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.WithValidation](10, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.WithValidation](10, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.WithValidation](10, utils.PseudoTx), gomock.Any()),

		// block 11
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.WithValidation](11, 3), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.WithValidation](11, 3), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.WithValidation](11, 3), gomock.Any()),

		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.WithValidation](11, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.WithValidation](11, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.WithValidation](11, utils.PseudoTx), gomock.Any()),

		ext.EXPECT().PostRun(executor.AtBlock[txcontext.WithValidation](12), gomock.Any(), nil),
	)

	if err := record(cfg, provider, processor, []executor.Extension[txcontext.WithValidation]{ext}); err != nil {
		t.Errorf("record failed: %v", err)
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

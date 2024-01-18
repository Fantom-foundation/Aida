package trace

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/txcontext"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

var testingAddress = common.Address{1}

func TestSdbReplaySubstate_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	processor := executor.NewMockProcessor[txcontext.TxContext](ctrl)
	ext := executor.NewMockExtension[txcontext.TxContext](ctrl)

	cfg := &utils.Config{}
	cfg.DbImpl = "carmen"
	cfg.KeepDb = false

	cfg.First = 0
	cfg.Last = 0

	provider.EXPECT().
		Run(0, 1, gomock.Any()).
		DoAndReturn(func(from int, to int, consumer executor.Consumer[txcontext.TxContext]) error {
			for i := from; i < to; i++ {
				consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 0, Transaction: 0, Data: substatecontext.NewTxContextWithValidation(testTx)})
				consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 0, Transaction: 1, Data: substatecontext.NewTxContextWithValidation(testTx)})
			}
			return nil
		})

	// All transactions are processed in order
	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[txcontext.TxContext](0), gomock.Any()),

		// tx 0
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](0, 0), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](0, 0), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](0, 0), gomock.Any()),

		// tx 1
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](0, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](0, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](0, 1), gomock.Any()),

		ext.EXPECT().PostRun(executor.AtBlock[txcontext.TxContext](1), gomock.Any(), nil),
	)

	if err := replaySubstate(cfg, provider, processor, nil, []executor.Extension[txcontext.TxContext]{ext}); err != nil {
		t.Errorf("record failed: %v", err)
	}
}

func TestSdbReplaySubstate_StateDbPrepperIsAddedIfDbImplIsMemory(t *testing.T) {
	ctrl := gomock.NewController(t)
	substateProvider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	operationProvider := executor.NewMockProvider[[]operation.Operation](ctrl)
	db := state.NewMockStateDB(ctrl)

	cfg := &utils.Config{}
	cfg.DbImpl = "memory"
	cfg.KeepDb = false

	cfg.First = 0
	cfg.Last = 0

	substateProvider.EXPECT().
		Run(0, 1, gomock.Any()).
		DoAndReturn(func(from int, to int, consumer executor.Consumer[txcontext.TxContext]) error {
			for i := from; i < to; i++ {
				consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 0, Transaction: 0, Data: substatecontext.NewTxContextWithValidation(testTx)})
			}
			return nil
		})
	operationProvider.EXPECT().
		Run(0, 0, gomock.Any()).
		DoAndReturn(func(from int, to int, consumer executor.Consumer[[]operation.Operation]) error {
			for i := from; i < to; i++ {
				consumer(executor.TransactionInfo[[]operation.Operation]{Block: 0, Transaction: 0, Data: testOperationsA})
			}
			return nil
		})

	processor := makeSubstateProcessor(cfg, context.NewReplay(), operationProvider)

	// if DbPrepper is added PrepareSubstate is called
	db.EXPECT().PrepareSubstate(gomock.Any(), uint64(0))

	if err := replaySubstate(cfg, substateProvider, processor, db, nil); err != nil {
		t.Errorf("record failed: %v", err)
	}
}

func TestSdbReplaySubstate_TxPrimerIsAddedIfDbImplIsNotMemory(t *testing.T) {
	ctrl := gomock.NewController(t)
	substateProvider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	operationProvider := executor.NewMockProvider[[]operation.Operation](ctrl)
	db := state.NewMockStateDB(ctrl)
	bulkLoad := state.NewMockBulkLoad(ctrl)

	cfg := &utils.Config{}
	cfg.DbImpl = "carmen"
	cfg.KeepDb = false

	cfg.First = 1
	cfg.Last = 1

	substateProvider.EXPECT().
		Run(1, 2, gomock.Any()).
		DoAndReturn(func(from int, to int, consumer executor.Consumer[txcontext.TxContext]) error {
			for i := from; i < to; i++ {
				consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 1, Transaction: 0, Data: substatecontext.NewTxContextWithValidation(testTx)})
			}
			return nil
		})
	operationProvider.EXPECT().
		Run(1, 1, gomock.Any()).
		DoAndReturn(func(from int, to int, consumer executor.Consumer[[]operation.Operation]) error {
			for i := from; i < to; i++ {
				consumer(executor.TransactionInfo[[]operation.Operation]{Block: 1, Transaction: 0, Data: testOperationsA})
			}
			return nil
		})

	processor := makeSubstateProcessor(cfg, context.NewReplay(), operationProvider)

	db.EXPECT().StartBulkLoad(uint64(0)).Return(bulkLoad)
	bulkLoad.EXPECT().Close()

	if err := replaySubstate(cfg, substateProvider, processor, db, nil); err != nil {
		t.Errorf("record failed: %v", err)
	}
}

var testOperationsA = []operation.Operation{
	operation.NewBeginBlock(0),
	operation.NewBeginTransaction(0),
	operation.NewExist(common.Address{}),
	operation.NewEndTransaction(),
}

var testOperationsB = []operation.Operation{
	operation.NewBeginTransaction(1),
	operation.NewExist(common.Address{}),
	operation.NewEndTransaction(),
	operation.NewEndBlock(),
}

// testTx is a dummy substate that will be processed without crashing.
var testTx = &substate.Substate{
	Env: &substate.SubstateEnv{},
	Message: &substate.SubstateMessage{
		Gas:      10000,
		GasPrice: big.NewInt(0),
	},
	Result: &substate.SubstateResult{
		GasUsed: 1,
	},
}

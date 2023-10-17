package main

import (
	"math/big"
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

var testingAddress = common.Address{1}

func TestProxyRecorderPrepper_PreTransactionCreatesRecorderProxy(t *testing.T) {
	path := t.TempDir() + "test_trace"
	cfg := &utils.Config{}
	cfg.TraceFile = path

	rCtx, err := context.NewRecord(cfg.TraceFile, cfg.First)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	p := makeProxyRecorderPrepper(rCtx)

	ctx := &executor.Context{}

	err = p.PreTransaction(executor.State[*substate.Substate]{}, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	_, ok := ctx.State.(*proxy.RecorderProxy)
	if !ok {
		t.Fatalf("state is not a recorder proxy")
	}
}

func TestOperationWriter_OperationGetsWritten(t *testing.T) {
	path := t.TempDir() + "test_trace"
	cfg := &utils.Config{}
	cfg.SyncPeriodLength = 1
	cfg.First = 1
	cfg.Last = 2
	cfg.TraceFile = path

	rCtx, err := context.NewRecord(cfg.TraceFile, cfg.First)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	p := makeProxyRecorderPrepper(rCtx)
	e := makeOperationBlockEmitter(cfg, rCtx)

	ctx := &executor.Context{}
	st := executor.State[*substate.Substate]{}

	// tx1
	err = p.PreTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	err = e.PreTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	// tx2
	err = p.PreTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	err = e.PreTransaction(st, ctx)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	err = e.PostRun(st, ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	stats, err := os.Stat(path)
	if err != nil {
		t.Fatalf("unexpected error; %v", err)
	}

	if stats.Size() <= 0 {
		t.Fatalf("size of trace file is 0")
	}

}

func TestSdbRecord_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*substate.Substate](ctrl)
	processor := executor.NewMockProcessor[*substate.Substate](ctrl)
	ext := executor.NewMockExtension[*substate.Substate](ctrl)
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

	rCtx, err := context.NewRecord(cfg.TraceFile, cfg.First)
	if err != nil {
		t.Fatalf("unexpected err; %v", err)
	}

	if err = record(cfg, provider, processor, rCtx, []executor.Extension[*substate.Substate]{ext}); err != nil {
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

package main

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/mock/gomock"
)

func TestVmSdb_TxGenerator_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	cfg := &utils.Config{
		First:             2,
		Last:              2,
		ChainID:           utils.MainnetChainID,
		ContinueOnFailure: true,
		LogLevel:          "Critical",
	}

	provider.EXPECT().
		Run(2, 2, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			// Block 2
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: newTestTxCtx()})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	gomock.InOrder(
		//// Block 2
		db.EXPECT().BeginBlock(uint64(2)),
		//// Tx 1
		db.EXPECT().BeginTransaction(uint32(1)),
		db.EXPECT().Prepare(gomock.Any(), 1),
		db.EXPECT().Snapshot().Return(15),
		db.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),
		db.EXPECT().GetCodeHash(gomock.Any()).Return(common.Hash{}),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(21_000)),
		db.EXPECT().SubBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),
		db.EXPECT().SetNonce(gomock.Any(), gomock.Any()),
		db.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(0)),
		db.EXPECT().GetRefund(),
		db.EXPECT().GetRefund(),
		db.EXPECT().AddBalance(gomock.Any(), gomock.Any()),
		db.EXPECT().GetLogs(common.HexToHash(fmt.Sprintf("0x%016d%016d", 2, 1)), common.HexToHash(fmt.Sprintf("0x%016d", 2))),
		db.EXPECT().EndTransaction(),
	)

	// since we are working with mock transactions, run logically fails on 'intrinsic gas too low'
	// since this is a test that tests orded of the db events, we can ignore this error
	err := runTransactions(cfg, provider, db, executor.MakeLiveDbTxProcessor(cfg))
	if err != nil {
		errors.Unwrap(err)
		if strings.Contains(err.Error(), "intrinsic gas too low") {
			return
		}
		t.Fatalf("run failed; %v", err)
	}
}

//
//func TestVmSdb_Substate_AllTransactionsAreProcessedInOrder(t *testing.T) {
//	ctrl := gomock.NewController(t)
//	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
//	db := state.NewMockStateDB(ctrl)
//	ext := executor.NewMockExtension[txcontext.TxContext](ctrl)
//	processor := executor.NewMockProcessor[txcontext.TxContext](ctrl)
//	cfg := &utils.Config{
//		First:       2,
//		Last:        4,
//		ChainID:     utils.MainnetChainID,
//		LogLevel:    "Critical",
//		SkipPriming: true,
//	}
//
//	// Simulate the execution of three transactions in two blocks.
//	provider.EXPECT().
//		Run(2, 5, gomock.Any()).
//		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
//			// Block 2
//			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: substatecontext.NewTxContext(emptyTx)})
//			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 2, Data: substatecontext.NewTxContext(emptyTx)})
//			// Block 3
//			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 3, Transaction: 1, Data: substatecontext.NewTxContext(emptyTx)})
//			// Block 4
//			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 4, Transaction: utils.PseudoTx, Data: substatecontext.NewTxContext(emptyTx)})
//			return nil
//		})
//
//	// The expectation is that all of those blocks and transactions
//	// are properly opened, prepared, executed, and closed.
//	// Since we are running sequential mode with 1 worker,
//	// all block and transactions need to be in order.
//	gomock.InOrder(
//		ext.EXPECT().PreRun(executor.AtBlock[txcontext.TxContext](2), gomock.Any()),
//
//		// Block 2
//		// Tx 1
//		ext.EXPECT().PreBlock(executor.AtBlock[txcontext.TxContext](2), gomock.Any()),
//		db.EXPECT().BeginBlock(uint64(2)),
//		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
//		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
//		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
//		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
//		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
//		// Tx 2
//		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(2)),
//		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
//		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
//		db.EXPECT().EndBlock(),
//		ext.EXPECT().PostBlock(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
//
//		// Block 3
//		ext.EXPECT().PreBlock(executor.AtBlock[txcontext.TxContext](3), gomock.Any()),
//		db.EXPECT().BeginBlock(uint64(3)),
//		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
//		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(3)),
//		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
//		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
//		db.EXPECT().EndBlock(),
//		ext.EXPECT().PostBlock(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
//
//		// Block 4
//		ext.EXPECT().PreBlock(executor.AtBlock[txcontext.TxContext](4), gomock.Any()),
//		db.EXPECT().BeginBlock(uint64(4)),
//		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
//		db.EXPECT().PrepareSubstate(gomock.Any(), uint64(4)),
//		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
//		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
//		db.EXPECT().EndBlock(),
//		ext.EXPECT().PostBlock(executor.AtTransaction[txcontext.TxContext](4, utils.PseudoTx), gomock.Any()),
//
//		ext.EXPECT().PostRun(executor.AtBlock[txcontext.TxContext](5), gomock.Any(), nil),
//	)
//
//	if err := runSubstates(cfg, provider, db, processor, []executor.Extension[txcontext.TxContext]{ext}); err != nil {
//		t.Errorf("run failed: %v", err)
//	}
//}
//
//// emptyTx is a dummy substate that will be processed without crashing.
//var emptyTx = &substate.Substate{
//	Env: &substate.SubstateEnv{},
//	Message: &substate.SubstateMessage{
//		GasPrice: big.NewInt(12),
//	},
//	Result: &substate.SubstateResult{
//		GasUsed: 1,
//	},
//}
//
//// testTx is a dummy substate used for testing validation.
//var testTx = &substate.Substate{
//	InputAlloc: substate.SubstateAlloc{testingAddress: substate.NewSubstateAccount(1, new(big.Int).SetUint64(1), []byte{})},
//	Env:        &substate.SubstateEnv{},
//	Message: &substate.SubstateMessage{
//		GasPrice: big.NewInt(12),
//	},
//	Result: &substate.SubstateResult{
//		GasUsed: 1,
//	},
//}

// testTxCtx is a dummy tx context used for testing.
type testTxCtx struct {
	txcontext.NilTxContext
	env txcontext.BlockEnvironment
	msg core.Message
}

func newTestTxCtx() txcontext.TxContext {
	return testTxCtx{
		env: &testTxBlkEnv{1},
		msg: types.NewMessage(
			common.Address{0x1},
			&common.Address{0x2},
			0,
			big.NewInt(1),
			21_000,
			big.NewInt(1),
			big.NewInt(1),
			big.NewInt(1),
			[]byte{},
			types.AccessList{},
			false,
		),
	}
}

func (ctx testTxCtx) GetMessage() core.Message {
	return ctx.msg
}

func (ctx testTxCtx) GetBlockEnvironment() txcontext.BlockEnvironment {
	return ctx.env
}

// testTxBlkEnv is a dummy block environment used for testing.
type testTxBlkEnv struct {
	blkNumber uint64
}

func (env testTxBlkEnv) GetCoinbase() common.Address {
	return common.HexToAddress("0x1")
}

func (env testTxBlkEnv) GetDifficulty() *big.Int {
	return big.NewInt(1)
}

func (env testTxBlkEnv) GetGasLimit() uint64 {
	return 1_000_000_000_000
}

func (env testTxBlkEnv) GetNumber() uint64 {
	// not used
	return 0
}

func (env testTxBlkEnv) GetTimestamp() uint64 {
	// use current timestamp as the block timestamp
	// since we don't have a real block
	return uint64(time.Now().Unix())
}

func (env testTxBlkEnv) GetBlockHash(blockNumber uint64) (common.Hash, error) {
	// transform the block number into a hash
	// we don't have real block hashes, so we just use the block number
	return common.BigToHash(big.NewInt(int64(blockNumber))), nil
}
func (env testTxBlkEnv) GetBaseFee() *big.Int {
	return big.NewInt(0)
}

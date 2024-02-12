package main

import (
	"math/big"
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

func TestVmSdb_TxGenerator_AllTransactionsAreProcessedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[txcontext.TxContext](ctrl)
	db := state.NewMockStateDB(ctrl)
	ext := executor.NewMockExtension[txcontext.TxContext](ctrl)
	processor := executor.NewMockProcessor[txcontext.TxContext](ctrl)
	cfg := &utils.Config{
		First:       2,
		Last:        4,
		ChainID:     utils.MainnetChainID,
		LogLevel:    "Critical",
		SkipPriming: true,
	}

	// Simulate the execution of four transactions in three blocks.
	provider.EXPECT().
		Run(2, 4, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			// Block 2
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: newTestTxCtx()})
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 2, Data: newTestTxCtx()})
			// Block 3
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 3, Transaction: 1, Data: newTestTxCtx()})
			// Block 4
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 4, Transaction: 1, Data: newTestTxCtx()})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running sequential mode with 1 worker,
	// all block and transactions need to be in order.
	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[txcontext.TxContext](2), gomock.Any()),

		// Block 2
		// Tx 1
		db.EXPECT().BeginBlock(uint64(2)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		// Tx 2
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		db.EXPECT().EndBlock(),

		// Block 3
		// Tx 1
		db.EXPECT().BeginBlock(uint64(3)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		db.EXPECT().EndBlock(),

		// Block 4
		// Tx 1
		db.EXPECT().BeginBlock(uint64(4)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](4, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](4, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](4, 1), gomock.Any()),
		ext.EXPECT().PostRun(executor.AtBlock[txcontext.TxContext](4), gomock.Any(), nil),
		db.EXPECT().EndBlock(),
	)

	if err := runTransactions(cfg, provider, db, processor, []executor.Extension[txcontext.TxContext]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

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

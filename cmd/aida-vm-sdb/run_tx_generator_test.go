// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"math/big"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/txcontext/txgenerator"
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

	cfg := utils.NewTestConfig(t, utils.MainnetChainID, 2, 4, false)
	// Simulate the execution of four transactions in three blocks.
	provider.EXPECT().
		Run(2, 4, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[txcontext.TxContext]) error {
			// Block 2
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 1, Data: newTestTxCtx(t, 2)})
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 2, Transaction: 2, Data: newTestTxCtx(t, 2)})
			// Block 3
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 3, Transaction: 1, Data: newTestTxCtx(t, 3)})
			// Block 4
			consumer(executor.TransactionInfo[txcontext.TxContext]{Block: 4, Transaction: 1, Data: newTestTxCtx(t, 4)})
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
		db.EXPECT().BeginTransaction(uint32(1)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 1), gomock.Any()),
		db.EXPECT().EndTransaction(),
		// Tx 2
		db.EXPECT().BeginTransaction(uint32(2)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](2, 2), gomock.Any()),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),

		// Block 3
		// Tx 1
		db.EXPECT().BeginBlock(uint64(3)),
		db.EXPECT().BeginTransaction(uint32(1)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](3, 1), gomock.Any()),
		db.EXPECT().EndTransaction(),
		db.EXPECT().EndBlock(),

		// Block 4
		// Tx 1
		db.EXPECT().BeginBlock(uint64(4)),
		db.EXPECT().BeginTransaction(uint32(1)),
		ext.EXPECT().PreTransaction(executor.AtTransaction[txcontext.TxContext](4, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[txcontext.TxContext](4, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[txcontext.TxContext](4, 1), gomock.Any()),
		db.EXPECT().EndTransaction(),
		ext.EXPECT().PostRun(executor.AtBlock[txcontext.TxContext](4), gomock.Any(), nil),
		db.EXPECT().EndBlock(),

		// db_manager closes the db
		db.EXPECT().Close(),
	)

	if err := runTransactions(cfg, provider, db, t.TempDir(), processor, []executor.Extension[txcontext.TxContext]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func newTestTxCtx(t *testing.T, blkNumber uint64) txcontext.TxContext {
	return txgenerator.NewTxContext(
		testTxBlkEnv{blkNumber},
		&core.Message{
			To:                &common.Address{0x2},
			From:              common.Address{0x1},
			Nonce:             0,
			Value:             big.NewInt(1),
			GasLimit:          21_000,
			GasPrice:          big.NewInt(1),
			GasFeeCap:         big.NewInt(1),
			GasTipCap:         big.NewInt(1),
			Data:              []byte{},
			AccessList:        types.AccessList{},
			SkipAccountChecks: false,
		},
	)
}

// testTxBlkEnv is a dummy block environment used for testing.
type testTxBlkEnv struct {
	blkNumber uint64
}

func (env testTxBlkEnv) GetRandom() *common.Hash {
	return nil
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
	return env.blkNumber
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

func (env testTxBlkEnv) GetBlobBaseFee() *big.Int {
	return big.NewInt(0)
}

func (env testTxBlkEnv) GetFork() string {
	return ""
}

package main

import (
	"errors"
	"math/rand"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/stochastic/generator"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

var simulation = &stochastic.EstimationModelJSON{
	FileId:           "1",
	Operations:       []string{},
	StochasticMatrix: [][]float64{{1.0}, {2.0}},
	Contracts: stochastic.EstimationStatsJSON{
		NumKeys:           generator.MinRandomAccessSize,
		Lambda:            1.1,
		QueueDistribution: []float64{1.0, 2.0},
	},
	Keys: stochastic.EstimationStatsJSON{
		NumKeys:           generator.MinRandomAccessSize,
		Lambda:            1.1,
		QueueDistribution: []float64{1.0, 2.0},
	},
	Values: stochastic.EstimationStatsJSON{
		NumKeys:           generator.MinRandomAccessSize,
		Lambda:            1.1,
		QueueDistribution: []float64{1.0, 2.0},
	},
	SnapshotLambda: 1,
}

var rg = rand.New(rand.NewSource(1))

func TestVmSdb_Substate_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[stochastic.Data](ctrl)
	db := state.NewMockStateDB(ctrl)
	cfg := &utils.Config{
		First:             2,
		Last:              4,
		ChainID:           utils.MainnetChainID,
		SkipPriming:       true,
		ContinueOnFailure: true,
		LogLevel:          "Critical",
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[stochastic.Data]) error {
			// Block 2
			consumer(executor.TransactionInfo[stochastic.Data]{Block: 2, Transaction: 1, Data: existData})
			consumer(executor.TransactionInfo[stochastic.Data]{Block: 2, Transaction: 2, Data: beginTransactionData})
			// Block 3
			consumer(executor.TransactionInfo[stochastic.Data]{Block: 3, Transaction: 1, Data: beginBlockData})
			// Block 4
			consumer(executor.TransactionInfo[stochastic.Data]{Block: 4, Transaction: utils.PseudoTx, Data: addBalanceData})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	gomock.InOrder(
		db.EXPECT().Exist(common.Address{byte(0)}),
		db.EXPECT().BeginTransaction(uint32(2)),
		db.EXPECT().BeginBlock(uint64(3)),
		db.EXPECT().AddBalance(common.Address{byte(0)}, executor.WithBigIntOfAnySize()),
	)

	// since we are working with mock transactions, run logically fails on 'intrinsic gas too low'
	// since this is a test that tests orded of the db events, we can ignore this error
	err := runStochasticReplay(cfg, provider, db, makeStochasticProcessor(cfg, simulation, rg), nil)
	if err != nil {
		errors.Unwrap(err)
		if strings.Contains(err.Error(), "intrinsic gas too low") {
			return
		}
		t.Fatal("run failed")
	}
}

func TestVmSdb_Substate_AllTransactionsAreProcessedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[stochastic.Data](ctrl)
	db := state.NewMockStateDB(ctrl)
	ext := executor.NewMockExtension[stochastic.Data](ctrl)
	processor := executor.NewMockProcessor[stochastic.Data](ctrl)
	cfg := &utils.Config{
		First:       2,
		Last:        4,
		ChainID:     utils.MainnetChainID,
		LogLevel:    "Critical",
		SkipPriming: true,
	}

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[stochastic.Data]) error {
			// Block 2
			consumer(executor.TransactionInfo[stochastic.Data]{Block: 2, Transaction: 1, Data: stochastic.Data{}})
			consumer(executor.TransactionInfo[stochastic.Data]{Block: 2, Transaction: 2, Data: stochastic.Data{}})
			// Block 3
			consumer(executor.TransactionInfo[stochastic.Data]{Block: 3, Transaction: 1, Data: stochastic.Data{}})
			// Block 4
			consumer(executor.TransactionInfo[stochastic.Data]{Block: 4, Transaction: utils.PseudoTx, Data: stochastic.Data{}})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running sequential mode with 1 worker,
	// all block and transactions need to be in order.
	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[stochastic.Data](2), gomock.Any()),

		// Block 2
		// Tx 1
		ext.EXPECT().PreBlock(executor.AtBlock[stochastic.Data](2), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[stochastic.Data](2, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[stochastic.Data](2, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[stochastic.Data](2, 1), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[stochastic.Data](2, 2), gomock.Any()),
		// Tx 2
		processor.EXPECT().Process(executor.AtTransaction[stochastic.Data](2, 2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[stochastic.Data](2, 2), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[stochastic.Data](2, 2), gomock.Any()),

		// Block 3
		ext.EXPECT().PreBlock(executor.AtBlock[stochastic.Data](3), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[stochastic.Data](3, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[stochastic.Data](3, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[stochastic.Data](3, 1), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[stochastic.Data](3, 1), gomock.Any()),

		// Block 4
		ext.EXPECT().PreBlock(executor.AtBlock[stochastic.Data](4), gomock.Any()),
		ext.EXPECT().PreTransaction(executor.AtTransaction[stochastic.Data](4, utils.PseudoTx), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[stochastic.Data](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[stochastic.Data](4, utils.PseudoTx), gomock.Any()),
		ext.EXPECT().PostBlock(executor.AtTransaction[stochastic.Data](4, utils.PseudoTx), gomock.Any()),

		ext.EXPECT().PostRun(executor.AtBlock[stochastic.Data](5), gomock.Any(), nil),
	)

	if err := runStochasticReplay(cfg, provider, db, processor, []executor.Extension[stochastic.Data]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

var beginBlockData = stochastic.Data{
	Operation: stochastic.BeginBlockID,
	Address:   0,
	Key:       0,
	Value:     0,
}

var beginTransactionData = stochastic.Data{
	Operation: stochastic.BeginTransactionID,
	Address:   0,
	Key:       0,
	Value:     0,
}

var existData = stochastic.Data{
	Operation: stochastic.ExistID,
	Address:   0,
	Key:       0,
	Value:     0,
}

var addBalanceData = stochastic.Data{
	Operation: stochastic.AddBalanceID,
	Address:   0,
	Key:       0,
	Value:     1,
}

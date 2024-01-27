package main

import (
	"math"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Norma/load/app"
	"github.com/urfave/cli/v2"
)

const testTreasureAccountPrivateKey = "1234567890123456789012345678901234567890123456789012345678901234"

// RunTxGenerator performs sequential block processing on a StateDb using transaction generator
func RunTxGenerator(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	cfg.StateValidationMode = utils.SubsetCheck

	cfg.DbImpl = "carmen"

	statedb, _, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}

	primaryAccount, err := app.NewAccount(0, testTreasureAccountPrivateKey, int64(cfg.ChainID))
	if err != nil {
		return err
	}

	_ = executor.MakeLiveDbTxProcessor(cfg)

	rpc := FakeRpcClient{stateDb: statedb}

	_, _ = app.NewCounterApplication(rpc, primaryAccount, 0, 0, 0)

	// todo init the provider (the generator) here and pass it to runTransactions

	return runTransactions(cfg, nil, statedb, executor.MakeLiveDbTxProcessor(cfg))
}

func runTransactions(
	cfg *utils.Config,
	provider executor.Provider[txcontext.TxContext],
	stateDb state.StateDB,
	processor executor.Processor[txcontext.TxContext],
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[txcontext.TxContext]{
		// todo choose extensions
	}

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:  0,
			To:    math.MaxInt,
			State: stateDb,
		},
		processor,
		extensionList,
	)
}

package main

import (
	"math"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

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

	provider := executor.NewNormaTxProvider(cfg, statedb)

	return runTransactions(cfg, provider, statedb, executor.MakeLiveDbTxProcessor(cfg))
}

func runTransactions(
	cfg *utils.Config,
	provider executor.Provider[txcontext.TxContext],
	stateDb state.StateDB,
	processor executor.Processor[txcontext.TxContext],
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[txcontext.TxContext]{
		//validator.MakeLiveDbValidator(cfg, validator.ValidateTxTarget{WorldState: false, Receipt: true}),
		// todo choose extensions
	}

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:                   0,
			To:                     math.MaxInt,
			State:                  stateDb,
			ParallelismGranularity: executor.TransactionLevel,
		},
		processor,
		extensionList,
	)
}

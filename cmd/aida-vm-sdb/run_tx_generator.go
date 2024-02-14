package main

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/logger"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/register"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// RunTxGenerator performs sequential block processing on a StateDb using transaction generator
func RunTxGenerator(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	cfg.StateValidationMode = utils.SubsetCheck

	db, _, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}

	provider := executor.NewNormaTxProvider(cfg, db)

	return runTransactions(cfg, provider, db, executor.MakeLiveDbTxProcessor(cfg), nil)
}

func runTransactions(
	cfg *utils.Config,
	provider executor.Provider[txcontext.TxContext],
	stateDb state.StateDB,
	processor executor.Processor[txcontext.TxContext],
	extra []executor.Extension[txcontext.TxContext],
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[txcontext.TxContext]{
		logger.MakeDbLogger[txcontext.TxContext](cfg),
		profiler.MakeVirtualMachineStatisticsPrinter[txcontext.TxContext](cfg),
		logger.MakeProgressLogger[txcontext.TxContext](cfg, 15*time.Second),
		logger.MakeErrorLogger[txcontext.TxContext](cfg),
		tracker.MakeBlockProgressTracker(cfg, 100),
		profiler.MakeMemoryUsagePrinter[txcontext.TxContext](cfg),
		profiler.MakeMemoryProfiler[txcontext.TxContext](cfg),
		statedb.MakeStateDbManager[txcontext.TxContext](cfg),

		register.MakeRegisterProgress(cfg, 100_000),
		// RegisterProgress should be the first on the list = last to receive PostRun.
		// This is because it collects the error and records it externally.
		// If not, error that happen afterwards (e.g. on top of) will not be correctly recorded.

		statedb.MakeTxGeneratorBlockEventEmitter[txcontext.TxContext](),
	}

	extensionList = append(extensionList, extra...)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:                   int(cfg.First),
			To:                     int(cfg.Last),
			State:                  stateDb,
			ParallelismGranularity: executor.TransactionLevel,
		},
		processor,
		extensionList,
	)
}

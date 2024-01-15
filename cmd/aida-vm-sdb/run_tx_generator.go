package main

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/primer"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/executor/transaction"
	"github.com/Fantom-foundation/Aida/state"
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

	// todo init the provider (the generator) here and pass it to runTransactions

	return runTransactions(cfg, nil, nil, false)
}

type txProcessor struct {
	cfg *utils.Config
}

func (p txProcessor) Process(state executor.State[transaction.SubstateData], ctx *executor.Context) error {
	// todo apply data onto StateDb
	return nil
}

func runTransactions(
	cfg *utils.Config,
	provider executor.Provider[transaction.SubstateData],
	stateDb state.StateDB,
	disableStateDbExtension bool,
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[transaction.SubstateData]{
		profiler.MakeCpuProfiler[transaction.SubstateData](cfg),
		profiler.MakeDiagnosticServer[transaction.SubstateData](cfg),
	}

	if !disableStateDbExtension {
		extensionList = append(
			extensionList,
			statedb.MakeStateDbManager[transaction.SubstateData](cfg),
			statedb.MakeLiveDbBlockChecker[transaction.SubstateData](cfg),
		)
	}

	extensionList = append(extensionList, []executor.Extension[transaction.SubstateData]{
		profiler.MakeThreadLocker[transaction.SubstateData](),
		profiler.MakeVirtualMachineStatisticsPrinter[transaction.SubstateData](cfg),
		tracker.MakeProgressLogger[transaction.SubstateData](cfg, 15*time.Second),
		//tracker.MakeProgressTracker(cfg, 100_000),
		primer.MakeStateDbPrimer[transaction.SubstateData](cfg),
		profiler.MakeMemoryUsagePrinter[transaction.SubstateData](cfg),
		profiler.MakeMemoryProfiler[transaction.SubstateData](cfg),
		//statedb.MakeStateDbPrepper(),
		//statedb.MakeArchiveInquirer(cfg),
		validator.MakeStateHashValidator[transaction.SubstateData](cfg),
		statedb.MakeBlockEventEmitter[transaction.SubstateData](),
		profiler.MakeOperationProfiler[transaction.SubstateData](cfg),
		// block profile extension should be always last because:
		// 1) Pre-Func are called forwards so this is called last and
		// 2) Post-Func are called backwards so this is called first
		// that means the gap between time measurements will be as small as possible
		//profiler.MakeBlockRuntimeAndGasCollector(cfg),
	}...,
	)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:  int(cfg.First),
			To:    int(cfg.Last) + 1,
			State: stateDb,
		},
		txProcessor{cfg},
		extensionList,
	)
}

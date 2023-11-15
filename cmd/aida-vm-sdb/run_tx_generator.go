package main

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

type GeneratedTransaction struct {
	// todo fill with transaction info from the generator
}

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

func (p txProcessor) Process(state executor.State[*GeneratedTransaction], ctx *executor.Context) error {
	// todo apply data onto StateDb
	return nil
}

func runTransactions(
	cfg *utils.Config,
	provider executor.Provider[*GeneratedTransaction],
	stateDb state.StateDB,
	disableStateDbExtension bool,
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[*GeneratedTransaction]{
		profiler.MakeCpuProfiler[*GeneratedTransaction](cfg),
		profiler.MakeDiagnosticServer[*GeneratedTransaction](cfg),
	}

	if !disableStateDbExtension {
		extensionList = append(extensionList, statedb.MakeStateDbManager[*GeneratedTransaction](cfg))
	}

	extensionList = append(extensionList, []executor.Extension[*GeneratedTransaction]{
		profiler.MakeThreadLocker[*GeneratedTransaction](),
		extension.MakeAidaDbManager[*GeneratedTransaction](cfg),
		profiler.MakeVirtualMachineStatisticsPrinter[*GeneratedTransaction](cfg),
		tracker.MakeProgressLogger[*GeneratedTransaction](cfg, 15*time.Second),
		//tracker.MakeProgressTracker(cfg, 100_000),
		statedb.MakeStateDbPrimer[*GeneratedTransaction](cfg),
		profiler.MakeMemoryUsagePrinter[*GeneratedTransaction](cfg),
		profiler.MakeMemoryProfiler[*GeneratedTransaction](cfg),
		//statedb.MakeStateDbPrepper(),
		//statedb.MakeArchiveInquirer(cfg),
		validator.MakeStateHashValidator[*GeneratedTransaction](cfg),
		statedb.MakeBlockEventEmitter[*GeneratedTransaction](),
		profiler.MakeOperationProfiler[*GeneratedTransaction](cfg),
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

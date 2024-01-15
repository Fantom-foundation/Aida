package main

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/aidadb"
	"github.com/Fantom-foundation/Aida/executor/extension/primer"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/register"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/executor/transaction"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// RunSubstate performs sequential block processing on a StateDb
func RunSubstate(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	cfg.StateValidationMode = utils.SubsetCheck

	substateDb, err := executor.OpenSubstateDb(cfg, ctx)
	if err != nil {
		return err
	}
	defer substateDb.Close()

	return runSubstates(cfg, substateDb, nil, executor.MakeLiveDbProcessor(cfg), nil)
}

func runSubstates(
	cfg *utils.Config,
	provider executor.Provider[transaction.SubstateData],
	stateDb state.StateDB,
	processor executor.Processor[transaction.SubstateData],
	extra []executor.Extension[transaction.SubstateData],
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[transaction.SubstateData]{
		profiler.MakeCpuProfiler[transaction.SubstateData](cfg),
		profiler.MakeDiagnosticServer[transaction.SubstateData](cfg),
	}

	if stateDb == nil {
		extensionList = append(
			extensionList,
			statedb.MakeStateDbManager[transaction.SubstateData](cfg),
			statedb.MakeLiveDbBlockChecker[transaction.SubstateData](cfg),
			tracker.MakeDbLogger[transaction.SubstateData](cfg),
		)
	}

	extensionList = append(extensionList, extra...)

	extensionList = append(extensionList, []executor.Extension[transaction.SubstateData]{
		register.MakeRegisterProgress(cfg, 100_000),
		// RegisterProgress should be the first on the list = last to receive PostRun.
		// This is because it collects the error and records it externally.
		// If not, error that happen afterwards (e.g. on top of) will not be correcly recorded.

		profiler.MakeThreadLocker[transaction.SubstateData](),
		aidadb.MakeAidaDbManager[transaction.SubstateData](cfg),
		profiler.MakeVirtualMachineStatisticsPrinter[transaction.SubstateData](cfg),
		tracker.MakeProgressLogger[transaction.SubstateData](cfg, 15*time.Second),
		tracker.MakeErrorLogger[transaction.SubstateData](cfg),
		tracker.MakeProgressTracker(cfg, 100_000),
		primer.MakeStateDbPrimer[transaction.SubstateData](cfg),
		profiler.MakeMemoryUsagePrinter[transaction.SubstateData](cfg),
		profiler.MakeMemoryProfiler[transaction.SubstateData](cfg),
		statedb.MakeStateDbPrepper(),
		statedb.MakeArchiveInquirer(cfg),
		validator.MakeStateHashValidator[transaction.SubstateData](cfg),
		statedb.MakeBlockEventEmitter[transaction.SubstateData](),
		validator.MakeLiveDbValidator(cfg),
		profiler.MakeOperationProfiler[transaction.SubstateData](cfg),

		// block profile extension should be always last because:
		// 1) Pre-Func are called forwards so this is called last and
		// 2) Post-Func are called backwards so this is called first
		// that means the gap between time measurements will be as small as possible
		profiler.MakeBlockRuntimeAndGasCollector(cfg),
	}...,
	)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:                   int(cfg.First),
			To:                     int(cfg.Last) + 1,
			NumWorkers:             1, // vm-sdb can run only with one worker
			State:                  stateDb,
			ParallelismGranularity: executor.BlockLevel,
		},
		processor,
		extensionList,
	)
}

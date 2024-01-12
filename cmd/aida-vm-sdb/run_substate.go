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
	provider executor.Provider[executor.TransactionData],
	stateDb state.StateDB,
	processor executor.Processor[executor.TransactionData],
	extra []executor.Extension[executor.TransactionData],
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[executor.TransactionData]{
		profiler.MakeCpuProfiler[executor.TransactionData](cfg),
		profiler.MakeDiagnosticServer[executor.TransactionData](cfg),
	}

	if stateDb == nil {
		extensionList = append(
			extensionList,
			statedb.MakeStateDbManager[executor.TransactionData](cfg),
			statedb.MakeLiveDbBlockChecker[executor.TransactionData](cfg),
			tracker.MakeDbLogger[executor.TransactionData](cfg),
		)
	}

	extensionList = append(extensionList, extra...)

	extensionList = append(extensionList, []executor.Extension[executor.TransactionData]{
		register.MakeRegisterProgress(cfg, 100_000),
		// RegisterProgress should be the first on the list = last to receive PostRun.
		// This is because it collects the error and records it externally.
		// If not, error that happen afterwards (e.g. on top of) will not be correcly recorded.

		profiler.MakeThreadLocker[executor.TransactionData](),
		aidadb.MakeAidaDbManager[executor.TransactionData](cfg),
		profiler.MakeVirtualMachineStatisticsPrinter[executor.TransactionData](cfg),
		tracker.MakeProgressLogger[executor.TransactionData](cfg, 15*time.Second),
		tracker.MakeErrorLogger[executor.TransactionData](cfg),
		tracker.MakeProgressTracker(cfg, 100_000),
		primer.MakeStateDbPrimer[executor.TransactionData](cfg),
		profiler.MakeMemoryUsagePrinter[executor.TransactionData](cfg),
		profiler.MakeMemoryProfiler[executor.TransactionData](cfg),
		statedb.MakeStateDbPrepper(),
		statedb.MakeArchiveInquirer(cfg),
		validator.MakeStateHashValidator[executor.TransactionData](cfg),
		statedb.MakeBlockEventEmitter[executor.TransactionData](),
		validator.MakeLiveDbValidator(cfg),
		profiler.MakeOperationProfiler[executor.TransactionData](cfg),

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

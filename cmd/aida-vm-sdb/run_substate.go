package main

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/aidadb"
	"github.com/Fantom-foundation/Aida/executor/extension/primer"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
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
	provider executor.Provider[*substate.Substate],
	stateDb state.StateDB,
	processor executor.Processor[*substate.Substate],
	extra []executor.Extension[*substate.Substate],
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[*substate.Substate]{
		profiler.MakeCpuProfiler[*substate.Substate](cfg),
		profiler.MakeDiagnosticServer[*substate.Substate](cfg),
	}

	if stateDb == nil {
		extensionList = append(
			extensionList,
			statedb.MakeStateDbManager[*substate.Substate](cfg),
			statedb.MakeLiveDbBlockChecker[*substate.Substate](cfg),
			tracker.MakeDbLogger[*substate.Substate](cfg),
		)
	}

	extensionList = append(extensionList, extra...)

	extensionList = append(extensionList, []executor.Extension[*substate.Substate]{
		profiler.MakeThreadLocker[*substate.Substate](),
		aidadb.MakeAidaDbManager[*substate.Substate](cfg),
		profiler.MakeVirtualMachineStatisticsPrinter[*substate.Substate](cfg),
		tracker.MakeProgressLogger[*substate.Substate](cfg, 15*time.Second),
		tracker.MakeErrorLogger[*substate.Substate](cfg),
		tracker.MakeProgressTracker(cfg, 100_000),
		primer.MakeStateDbPrimer[*substate.Substate](cfg),
		profiler.MakeMemoryUsagePrinter[*substate.Substate](cfg),
		profiler.MakeMemoryProfiler[*substate.Substate](cfg),
		statedb.MakeStateDbPrepper(),
		statedb.MakeArchiveInquirer(cfg),
		validator.MakeStateHashValidator[*substate.Substate](cfg),
		statedb.MakeBlockEventEmitter[*substate.Substate](),
		validator.MakeLiveDbValidator(cfg),

		profiler.MakeOperationProfiler[*substate.Substate](cfg),
		// block profile extension should be always last because:
		// 1) Pre-Func are called forwards so this is called last and
		// 2) Post-Func are called backwards so this is called first
		// that means the gap between time measurements will be as small as possible
		profiler.MakeBlockRuntimeAndGasCollector(cfg),
	}...,
	)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:  int(cfg.First),
			To:    int(cfg.Last) + 1,
			State: stateDb,
		},
		processor,
		extensionList,
	)
}

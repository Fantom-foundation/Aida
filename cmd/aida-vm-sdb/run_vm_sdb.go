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
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// RunVmSdb performs sequential block processing on a StateDb
func RunVmSdb(ctx *cli.Context) error {
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

	return run(cfg, substateDb, nil, false)
}

type txProcessor struct {
	cfg *utils.Config
}

func (r txProcessor) Process(state executor.State[*substate.Substate], ctx *executor.Context) error {
	_, err := utils.ProcessTx(
		ctx.State,
		r.cfg,
		uint64(state.Block),
		state.Transaction,
		state.Data,
	)
	return err
}

func run(
	cfg *utils.Config,
	provider executor.Provider[*substate.Substate],
	stateDb state.StateDB,
	disableStateDbExtension bool,
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[*substate.Substate]{
		profiler.MakeCpuProfiler[*substate.Substate](cfg),
		profiler.MakeDiagnosticServer[*substate.Substate](cfg),
	}

	if !disableStateDbExtension {
		extensionList = append(extensionList, statedb.MakeStateDbManager[*substate.Substate](cfg))
	}

	extensionList = append(extensionList, []executor.Extension[*substate.Substate]{
		extension.MakeAidaDbManager[*substate.Substate](cfg),
		profiler.MakeVirtualMachineStatisticsPrinter[*substate.Substate](cfg),
		tracker.MakeProgressLogger[*substate.Substate](cfg, 15*time.Second),
		tracker.MakeProgressTracker(cfg, 100_000),
		statedb.MakeStateDbPrimer[*substate.Substate](cfg),
		profiler.MakeMemoryUsagePrinter[*substate.Substate](cfg),
		profiler.MakeMemoryProfiler[*substate.Substate](cfg),
		statedb.MakeStateDbPrepper(),
		validator.MakeStateHashValidator[*substate.Substate](cfg),
		statedb.MakeBlockEventEmitter[*substate.Substate](),
		profiler.MakeOperationProfiler[*substate.Substate](cfg),
	}...,
	)

	return executor.NewExecutor(provider).Run(
		executor.Params{
			From:  int(cfg.First),
			To:    int(cfg.Last) + 1,
			State: stateDb,
		},
		txProcessor{cfg},
		extensionList,
	)
}

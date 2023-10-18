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
	config *utils.Config
}

func (r txProcessor) Process(state executor.State[*substate.Substate], context *executor.Context) error {
	_, err := utils.ProcessTx(
		context.State,
		r.config,
		uint64(state.Block),
		state.Transaction,
		state.Data,
	)
	return err
}

func run(config *utils.Config, provider executor.Provider[*substate.Substate], stateDb state.StateDB, disableStateDbExtension bool) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[*substate.Substate]{
		profiler.MakeCpuProfiler[*substate.Substate](config),
		profiler.MakeDiagnosticServer[*substate.Substate](config),
	}

	if !disableStateDbExtension {
		extensionList = append(extensionList, statedb.MakeStateDbManager[*substate.Substate](config))
	}

	extensionList = append(extensionList, []executor.Extension[*substate.Substate]{
		extension.MakeAidaDbManager[*substate.Substate](config),
		profiler.MakeVirtualMachineStatisticsPrinter[*substate.Substate](config),
		tracker.MakeProgressLogger[*substate.Substate](config, 15*time.Second),
		tracker.MakeProgressTracker(config, 100_000),
		statedb.MakeStateDbPrimer[*substate.Substate](config),
		profiler.MakeMemoryUsagePrinter[*substate.Substate](config),
		profiler.MakeMemoryProfiler[*substate.Substate](config),
		statedb.MakeStateDbPrepper(),
		validator.MakeStateHashValidator[*substate.Substate](config),
		statedb.MakeBlockEventEmitter[*substate.Substate](),
		profiler.MakeOperationProfiler[*substate.Substate](config),
	}...,
	)

	return executor.NewExecutor(provider).Run(
		executor.Params{
			From:  int(config.First),
			To:    int(config.Last) + 1,
			State: stateDb,
		},
		txProcessor{config},
		extensionList,
	)
}

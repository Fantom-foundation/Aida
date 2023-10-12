package main

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
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

func (r txProcessor) Process(state executor.State, context *executor.Context) error {
	_, err := utils.ProcessTx(
		context.State,
		r.config,
		uint64(state.Block),
		state.Transaction,
		state.Substate,
	)
	return err
}

func run(config *utils.Config, provider executor.SubstateProvider, stateDb state.StateDB, disableStateDbExtension bool) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension{
		profiler.MakeCpuProfiler(config),
		profiler.MakeDiagnosticServer(config),
	}

	if !disableStateDbExtension {
		extensionList = append(extensionList, statedb.MakeStateDbManager(config))
	}

	extensionList = append(extensionList, []executor.Extension{
		profiler.MakeVirtualMachineStatisticsPrinter(config),
		tracker.MakeProgressLogger(config, 15*time.Second),
		tracker.MakeProgressTracker(config, 100_000),
		statedb.MakeStateDbPrimer(config),
		profiler.MakeMemoryUsagePrinter(config),
		profiler.MakeMemoryProfiler(config),
		statedb.MakeStateDbPrepper(),
		validator.MakeStateHashValidator(config),
		statedb.MakeBlockEventEmitter(),
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

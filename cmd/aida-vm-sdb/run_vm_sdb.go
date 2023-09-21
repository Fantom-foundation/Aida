package main

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
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
	var extensionList = []executor.Extension{extension.MakeCpuProfiler(config)}

	if !disableStateDbExtension {
		extensionList = append(extensionList, extension.MakeStateDbManager(config))
	}

	extensionList = append(extensionList, []executor.Extension{
		extension.MakeVirtualMachineStatisticsPrinter(config),
		extension.MakeProgressLogger(config, 15*time.Second),
		extension.MakeProgressTracker(config, 100_000),
		extension.MakeStateDbPrimer(config),
		extension.MakeMemoryUsagePrinter(config),
		extension.MakeMemoryProfiler(config),
		extension.MakeStateDbPreparator(),
		extension.MakeStateHashValidator(config),
		extension.MakeBlockEventEmitter(),
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

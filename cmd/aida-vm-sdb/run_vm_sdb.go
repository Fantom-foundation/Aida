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

	substateDb, err := executor.OpenSubstateDb(cfg, ctx)
	if err != nil {
		return err
	}
	defer substateDb.Close()

	stateDb, _, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}
	defer stateDb.Close()

	return run(cfg, substateDb, stateDb)
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

func run(config *utils.Config, provider executor.ActionProvider, stateDb state.StateDB) error {
	return executor.NewExecutor(provider).Run(
		executor.Params{
			From:  int(config.First),
			To:    int(config.Last) + 1,
			State: stateDb,
		},
		txProcessor{config},
		[]executor.Extension{
			extension.MakeCpuProfiler(config),
			extension.MakeVirtualMachineStatisticsPrinter(config),
			extension.MakeProgressLogger(config, 15*time.Second),
			extension.MakeProgressTracker(config, 100_000),
			extension.MakeStateDbPreparator(),
			extension.MakeTxValidator(config),
			extension.MakeStateHashValidator(config),
			extension.MakeBlockEventEmitter(),
		},
	)
}

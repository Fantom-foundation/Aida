package main

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// RunVmAdb performs block processing on an ArchiveDb
func RunVmAdb(ctx *cli.Context) error {
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

func (r txProcessor) Process(state executor.State) error {
	archive, err := state.State.GetArchiveState(uint64(state.Block))
	if err != nil {
		return err
	}
	_, err = utils.ProcessTx(
		archive,
		r.config,
		uint64(state.Block),
		state.Transaction,
		state.Substate,
	)
	return err
}

func run(config *utils.Config, provider executor.SubstateProvider, stateDb state.StateDB) error {
	return executor.NewExecutor(provider).Run(
		executor.Params{
			From:       int(config.First),
			To:         int(config.Last) + 1,
			State:      stateDb,
			NumWorkers: config.Workers,
		},
		txProcessor{config},
		[]executor.Extension{
			extension.MakeCpuProfiler(config),
			extension.MakeProgressLogger(config, 100),
			extension.MakeStateDbPreparator(),
			extension.MakeTxValidator(config),
			extension.MakeBeginOnlyEmitter(),
		},
	)
}

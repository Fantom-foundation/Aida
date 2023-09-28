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

	// executing archive blocks always calls ArchiveDb with block -1
	// this condition prevents an incorrect call for block that does not exist (block number -1 in this case)
	// there is nothing before block 0 so running this app on this block does nothing
	if cfg.First == 0 {
		cfg.First = 1
	}

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
		extension.MakeProgressLogger(config, 0),
		extension.MakeStateDbPreparator(),
		extension.MakeBeginOnlyEmitter(),
	}...)
	return executor.NewExecutor(provider).Run(
		executor.Params{
			From:          int(config.First),
			To:            int(config.Last) + 1,
			State:         stateDb,
			NumWorkers:    config.Workers,
			ExecutionType: executor.BlockIsolatedArchive,
		},
		txProcessor{config},
		extensionList,
	)
}

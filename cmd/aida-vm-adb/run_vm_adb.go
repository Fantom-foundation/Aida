package main

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler_extensions"
	"github.com/Fantom-foundation/Aida/executor/extension/progress_extensions"
	"github.com/Fantom-foundation/Aida/executor/extension/state_db_extensions"
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

	cfg.SrcDbReadonly = true

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

	stateDb, _, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}
	defer func() error {
		return stateDb.Close()
	}()

	return run(cfg, substateDb, stateDb, blockProcessor{cfg}, nil)
}

type blockProcessor struct {
	config *utils.Config
}

func (r blockProcessor) Process(state executor.State, context *executor.Context) error {
	_, err := utils.ProcessTx(
		context.Archive,
		r.config,
		uint64(state.Block),
		state.Transaction,
		state.Substate,
	)
	return err
}

func run(
	config *utils.Config,
	provider executor.SubstateProvider,
	stateDb state.StateDB,
	processor executor.Processor,
	extra []executor.Extension,
) error {
	extensionList := []executor.Extension{
		profiler_extensions.MakeCpuProfiler(config),
		state_db_extensions.MakeArchivePrepper(),
		progress_extensions.MakeProgressLogger(config, 100),
		state_db_extensions.MakeStateDbPrepper(),
	}
	extensionList = append(extensionList, extra...)
	return executor.NewExecutor(provider).Run(
		executor.Params{
			From:                   int(config.First),
			To:                     int(config.Last) + 1,
			State:                  stateDb,
			NumWorkers:             config.Workers,
			ParallelismGranularity: executor.BlockLevel,
		},
		processor,
		extensionList,
	)
}

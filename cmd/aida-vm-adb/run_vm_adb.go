package main

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/executor/transaction/substate_transaction"
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
	cfg.StateValidationMode = utils.SubsetCheck

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

	return run(cfg, substateDb, nil, executor.MakeArchiveDbProcessor(cfg), nil)
}

func run(
	cfg *utils.Config,
	provider executor.Provider[substate_transaction.SubstateData],
	stateDb state.StateDB,
	processor executor.Processor[substate_transaction.SubstateData],
	extra []executor.Extension[substate_transaction.SubstateData],
) error {
	extensionList := []executor.Extension[substate_transaction.SubstateData]{
		profiler.MakeCpuProfiler[substate_transaction.SubstateData](cfg),
		statedb.MakeArchivePrepper(),
		tracker.MakeProgressLogger[substate_transaction.SubstateData](cfg, 0),
		tracker.MakeErrorLogger[substate_transaction.SubstateData](cfg),
		validator.MakeArchiveDbValidator(cfg),
	}

	if stateDb == nil {
		extensionList = append(
			extensionList,
			statedb.MakeStateDbManager[substate_transaction.SubstateData](cfg),
			statedb.MakeArchiveBlockChecker[substate_transaction.SubstateData](cfg),
			tracker.MakeDbLogger[substate_transaction.SubstateData](cfg),
		)
	}

	extensionList = append(extensionList, extra...)
	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:                   int(cfg.First),
			To:                     int(cfg.Last) + 1,
			State:                  stateDb,
			NumWorkers:             cfg.Workers,
			ParallelismGranularity: executor.BlockLevel,
		},
		processor,
		extensionList,
	)
}

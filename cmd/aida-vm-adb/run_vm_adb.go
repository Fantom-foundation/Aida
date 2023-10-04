package main

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
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

func (r txProcessor) Process(state executor.State[*substate.Substate], context *executor.Context) error {
	// todo rework this once executor.State is divided between mutable and immutable part
	archive, err := context.State.GetArchiveState(uint64(state.Block) - 1)
	if err != nil {
		return err
	}
	_, err = utils.ProcessTx(
		archive,
		r.config,
		uint64(state.Block),
		state.Transaction,
		state.Payload,
	)
	return err
}

func run(config *utils.Config, provider executor.Provider[*substate.Substate], stateDb state.StateDB, disableStateDbExtension bool) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[*substate.Substate]{extension.MakeCpuProfiler[*substate.Substate](config)}

	if !disableStateDbExtension {
		extensionList = append(extensionList, extension.MakeStateDbManager[*substate.Substate](config))
	}

	extensionList = append(extensionList, []executor.Extension[*substate.Substate]{
		extension.MakeProgressLogger[*substate.Substate](config, 0),
		extension.MakeStateDbPreparator(),
		extension.MakeBeginOnlyEmitter[*substate.Substate](),
	}...)
	return executor.NewExecutor(provider).Run(
		executor.Params{
			From:       int(config.First),
			To:         int(config.Last) + 1,
			State:      stateDb,
			NumWorkers: config.Workers,
		},
		txProcessor{config},
		extensionList,
	)
}

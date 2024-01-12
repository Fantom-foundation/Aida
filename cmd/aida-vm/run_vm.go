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

// RunVm runs a range of transactions on an EVM in parallel.
func RunVm(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	substateDb, err := executor.OpenSubstateDb(cfg, ctx)
	if err != nil {
		return err
	}
	defer substateDb.Close()

	return run(cfg, substateDb, nil, executor.MakeLiveDbProcessor(cfg), nil)
}

// run executes the actual block-processing evaluation for RunVm above.
// It is factored out to facilitate testing without the need to create
// a cli.Context or to provide an actual SubstateDb on disk.
// Run defines the full set of executor extensions that are active in
// aida-vm and allows to define extra extensions for observing the
// execution, in particular during unit tests.
func run(
	cfg *utils.Config,
	provider executor.Provider[executor.TransactionData],
	stateDb state.StateDB,
	processor executor.Processor[executor.TransactionData],
	extra []executor.Extension[executor.TransactionData],
) error {
	extensions := []executor.Extension[executor.TransactionData]{
		profiler.MakeCpuProfiler[executor.TransactionData](cfg),
		profiler.MakeDiagnosticServer[executor.TransactionData](cfg),
		profiler.MakeVirtualMachineStatisticsPrinter[executor.TransactionData](cfg),
	}

	if stateDb == nil {
		extensions = append(
			extensions,
			statedb.MakeTemporaryStatePrepper(cfg),
			tracker.MakeDbLogger[executor.TransactionData](cfg),
		)
	}

	extensions = append(
		extensions,
		tracker.MakeErrorLogger[executor.TransactionData](cfg),
		tracker.MakeProgressLogger[executor.TransactionData](cfg, 15*time.Second),
		validator.MakeLiveDbValidator(cfg),
	)
	extensions = append(extensions, extra...)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:                   int(cfg.First),
			To:                     int(cfg.Last) + 1,
			NumWorkers:             cfg.Workers,
			State:                  stateDb,
			ParallelismGranularity: executor.TransactionLevel,
		},
		processor,
		extensions,
	)
}

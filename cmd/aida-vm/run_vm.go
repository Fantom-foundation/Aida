package main

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
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

	log := logger.NewLogger(cfg.LogLevel, "aida-vm")
	workers := cfg.Workers
	if workers <= 1 {
		workers = 1
	}
	log.Infof("Processing transactions using %d workers (--workers)...", workers)

	return run(cfg, substateDb, txProcessor{cfg}, nil)
}

// run executes the actual block-processing evaluation for RunVm above.
// It is factored out to facilitate testing without the need to create
// a cli.Context or to provide an actual SubstateDb on disk.
// Run defines the full set of executor extensions that are active in
// aida-vm and allows to define extra extensions for observing the
// execution, in particular during unit tests.
func run(
	config *utils.Config,
	provider executor.Provider[*substate.Substate],
	processor executor.Processor[*substate.Substate],
	extra []executor.Extension[*substate.Substate],
) error {
	extensions := []executor.Extension[*substate.Substate]{
		profiler.MakeCpuProfiler[*substate.Substate](config),
		profiler.MakeDiagnosticServer[*substate.Substate](config),
		profiler.MakeVirtualMachineStatisticsPrinter[*substate.Substate](config),
		tracker.MakeProgressLogger[*substate.Substate](config, 15*time.Second),
		statedb.MakeTemporaryStatePrepper(),
	}
	extensions = append(extensions, extra...)

	return executor.NewExecutor(provider).Run(
		executor.Params{
			From:       int(config.First),
			To:         int(config.Last) + 1,
			NumWorkers: config.Workers,
		},
		processor,
		extensions,
	)
}

type txProcessor struct {
	config *utils.Config
}

func (r txProcessor) Process(s executor.State[*substate.Substate], c *executor.Context) error {
	_, err := utils.ProcessTx(
		c.State,
		r.config,
		uint64(s.Block),
		s.Transaction,
		s.Data,
	)
	return err
}

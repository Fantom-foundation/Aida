package main

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/action_provider"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
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

	substateDb, err := action_provider.OpenSubstateDb(cfg, ctx)
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
	provider action_provider.ActionProvider,
	processor executor.Processor,
	extra []executor.Extension,
) error {
	extensions := []executor.Extension{
		extension.MakeCpuProfiler(config),
		extension.MakeVirtualMachineStatisticsPrinter(config),
		extension.MakeProgressLogger(config, 15*time.Second),
		temporaryStatePrepper{},
		extension.MakeTxValidator(config),
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

func (r txProcessor) Process(s executor.State, c *executor.Context) error {
	_, err := utils.ProcessTx(
		c.State,
		r.config,
		uint64(s.Block),
		s.Transaction,
		s.Substate,
	)
	return err
}

// temporaryStatePrepper is an extension that introduces a fresh in-memory
// StateDB instance before each transaction execution.
type temporaryStatePrepper struct {
	extension.NilExtension
}

func (temporaryStatePrepper) PreTransaction(s executor.State, c *executor.Context) error {
	c.State = state.MakeInMemoryStateDB(&s.Substate.InputAlloc, uint64(s.Block))
	return nil
}

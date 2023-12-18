package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/aidadb"
	"github.com/Fantom-foundation/Aida/executor/extension/primer"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// StochasticReplayCommand data structure for the replay app.
var StochasticReplayCommand = cli.Command{
	Action:    RunStochasticReplay,
	Name:      "replay",
	Usage:     "Simulates StateDB operations using a random generator with realistic distributions",
	ArgsUsage: "<simulation-length> <simulation-file>",
	Flags: []cli.Flag{
		&utils.BalanceRangeFlag,
		&utils.CarmenSchemaFlag,
		&utils.ContinueOnFailureFlag,
		&utils.CpuProfileFlag,
		&utils.DebugFromFlag,
		&utils.MemoryBreakdownFlag,
		&utils.NonceRangeFlag,
		&utils.RandomSeedFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.DbTmpFlag,
		&utils.StateDbLoggingFlag,
		&utils.TraceFileFlag,
		&utils.TraceDebugFlag,
		&utils.TraceFlag,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The stochastic replay command requires two argument:
<simulation-length> <simulation.json> 

<simulation-length> determines the number of blocks
<simulation.json> contains the simulation parameters produced by the stochastic estimator.`,
}

func RunStochasticReplay(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	if cfg.StochasticSimulationFile == "" {
		return fmt.Errorf("you must define path to simulation file (--%v)", utils.StochasticSimulationFileFlag.Name)
	}

	simulation, err := stochastic.ReadSimulation(cfg.StochasticSimulationFile)
	if err != nil {
		return fmt.Errorf("cannot read simulation; %v", err)
	}

	rg := rand.New(rand.NewSource(cfg.RandomSeed))

	simulations, err := executor.OpenSimulations(simulation, ctx, rg)
	if err != nil {
		return err
	}
	defer simulations.Close()

	return runStochasticReplay(cfg, simulations, nil, makeStochasticProcessor(cfg, simulation, rg), nil)

}

func makeStochasticProcessor(cfg *utils.Config, e *stochastic.EstimationModelJSON, rg *rand.Rand) executor.Processor[stochastic.Data] {
	return stochasticProcessor{
		stochastic.CreateState(e, rg, logger.NewLogger(cfg.LogLevel, "Stochastic Processor")), cfg,
	}
}

type stochasticProcessor struct {
	*stochastic.State
	cfg *utils.Config
}

func (p stochasticProcessor) Process(state executor.State[stochastic.Data], ctx *executor.Context) error {
	if p.cfg.Debug && state.Block >= p.cfg.DebugFrom {
		p.EnableDebug()
	}

	p.Execute(state.Block, state.Transaction, state.Data, ctx.State)
	return nil
}

func runStochasticReplay(
	cfg *utils.Config,
	provider executor.Provider[stochastic.Data],
	stateDb state.StateDB,
	processor executor.Processor[stochastic.Data],
	extra []executor.Extension[stochastic.Data],
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[stochastic.Data]{
		profiler.MakeCpuProfiler[stochastic.Data](cfg),
		profiler.MakeDiagnosticServer[stochastic.Data](cfg),
	}

	if stateDb == nil {
		extensionList = append(
			extensionList,
			statedb.MakeStateDbManager[stochastic.Data](cfg),
			tracker.MakeDbLogger[stochastic.Data](cfg),
		)
	}

	extensionList = append(extensionList, extra...)

	extensionList = append(extensionList, []executor.Extension[stochastic.Data]{
		profiler.MakeThreadLocker[stochastic.Data](),
		aidadb.MakeAidaDbManager[stochastic.Data](cfg),
		profiler.MakeVirtualMachineStatisticsPrinter[stochastic.Data](cfg),
		tracker.MakeProgressLogger[stochastic.Data](cfg, 15*time.Second),
		tracker.MakeErrorLogger[stochastic.Data](cfg),
		primer.MakeStateDbPrimer[stochastic.Data](cfg),
		profiler.MakeMemoryUsagePrinter[stochastic.Data](cfg),
		profiler.MakeMemoryProfiler[stochastic.Data](cfg),
		validator.MakeStateHashValidator[stochastic.Data](cfg),

		profiler.MakeOperationProfiler[stochastic.Data](cfg),
	}...,
	)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:                   int(cfg.First),
			To:                     int(cfg.Last) + 1,
			NumWorkers:             1, // stochastic can run only with one worker
			State:                  stateDb,
			ParallelismGranularity: executor.BlockLevel,
		},
		processor,
		extensionList,
	)
}

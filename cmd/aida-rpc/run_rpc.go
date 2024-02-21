package main

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/logger"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/register"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

const (
	Rpc_DefaultProgressReportFrequency = 100_000
)

func RunRpc(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	cfg.SrcDbReadonly = true

	rpcSource, err := executor.OpenRpcRecording(cfg, ctx)
	if err != nil {
		return err
	}

	defer rpcSource.Close()

	return run(cfg, rpcSource, nil, makeRpcProcessor(cfg), nil)
}

func makeRpcProcessor(cfg *utils.Config) rpcProcessor {
	return rpcProcessor{
		cfg: cfg,
	}
}

type rpcProcessor struct {
	cfg *utils.Config
}

func (p rpcProcessor) Process(state executor.State[*rpc.RequestAndResults], ctx *executor.Context) error {
	state.Data.StateDB = rpc.Execute(uint64(state.Block), state.Data, ctx.Archive, p.cfg)
	return nil
}

func run(
	cfg *utils.Config,
	provider executor.Provider[*rpc.RequestAndResults],
	stateDb state.StateDB,
	processor executor.Processor[*rpc.RequestAndResults],
	extra []executor.Extension[*rpc.RequestAndResults],

) error {
	var extensionList = []executor.Extension[*rpc.RequestAndResults]{
		// RegisterProgress should be the first on the list = last to receive PostRun.
		// This is because it collects the error and records it externally.
		// If not, error that happen afterwards (e.g. on top of) will not be correcly recorded.
		register.MakeRegisterRequestProgress(cfg, Rpc_DefaultProgressReportFrequency),
		profiler.MakeCpuProfiler[*rpc.RequestAndResults](cfg),
		logger.MakeProgressLogger[*rpc.RequestAndResults](cfg, 15*time.Second),
		logger.MakeErrorLogger[*rpc.RequestAndResults](cfg),
		tracker.MakeRequestProgressTracker(cfg, 100_000),
		statedb.MakeTemporaryArchivePrepper(),
		validator.MakeRpcComparator(cfg),
	}

	// this is for testing purposes so mock statedb and mock extension can be used
	extensionList = append(extensionList, extra...)
	if stateDb == nil {
		extensionList = append(
			extensionList,
			statedb.MakeStateDbManager[*rpc.RequestAndResults](cfg, ""),
			statedb.MakeArchiveBlockChecker[*rpc.RequestAndResults](cfg),
			logger.MakeDbLogger[*rpc.RequestAndResults](cfg),
		)

	}

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:                   int(cfg.First),
			To:                     int(cfg.Last) + 1,
			NumWorkers:             cfg.Workers,
			ParallelismGranularity: executor.TransactionLevel,
			State:                  stateDb,
		},
		processor,
		extensionList,
	)
}

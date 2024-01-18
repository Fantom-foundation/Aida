package main

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// RecordCommand data structure for the record app
var RecordCommand = cli.Command{
	Action:    RecordStateDbTrace,
	Name:      "record",
	Usage:     "captures and records StateDB operations while processing blocks",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.UpdateBufferSizeFlag,
		&utils.CpuProfileFlag,
		&utils.SyncPeriodLengthFlag,
		&substate.WorkersFlag,
		&utils.ChainIDFlag,
		&utils.TraceFileFlag,
		&utils.TraceDebugFlag,
		&utils.DebugFromFlag,
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The trace record command requires two arguments:
<blockNumFirst> <blockNumLast>
<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to trace transactions.`,
}

func RecordStateDbTrace(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	// force enable txcontext validation
	cfg.ValidateTxState = true

	substate.RecordReplay = true
	substateDb, err := executor.OpenSubstateDb(cfg, ctx)
	if err != nil {
		return err
	}
	defer substateDb.Close()

	return record(cfg, substateDb, executor.MakeLiveDbProcessor(cfg), nil)
}

func record(
	cfg *utils.Config,
	provider executor.Provider[txcontext.WithValidation],
	processor executor.Processor[txcontext.WithValidation],
	extra []executor.Extension[txcontext.WithValidation],
) error {
	var extensions = []executor.Extension[txcontext.WithValidation]{
		profiler.MakeCpuProfiler[txcontext.WithValidation](cfg),
		tracker.MakeProgressLogger[txcontext.WithValidation](cfg, 0),
		tracker.MakeProgressTracker(cfg, 0),
		statedb.MakeTemporaryStatePrepper(cfg),
		statedb.MakeProxyRecorderPrepper(cfg),
		validator.MakeLiveDbValidator(cfg),
	}

	extensions = append(extensions, extra...)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:                   int(cfg.First),
			To:                     int(cfg.Last) + 1,
			ParallelismGranularity: executor.TransactionLevel,
		},
		processor,
		extensions,
	)
}

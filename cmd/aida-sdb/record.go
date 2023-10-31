package main

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/logger"
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
		&utils.QuietFlag,
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

	// force enable transaction validation
	cfg.ValidateTxState = true

	substate.RecordReplay = true
	substateDb, err := executor.OpenSubstateDb(cfg, ctx)
	if err != nil {
		return err
	}
	defer substateDb.Close()

	rec := newRecorder(cfg)

	return record(cfg, substateDb, rec, nil)
}

func newRecorder(cfg *utils.Config) *recorder {
	return &recorder{
		cfg: cfg,
	}
}

type recorder struct {
	cfg *utils.Config
}

func (r *recorder) Process(state executor.State[*substate.Substate], context *executor.Context) error {
	_, err := utils.ProcessTx(
		context.State,
		r.cfg,
		uint64(state.Block),
		state.Transaction,
		state.Data,
	)

	return err
}

func record(
	cfg *utils.Config,
	provider executor.Provider[*substate.Substate],
	processor executor.Processor[*substate.Substate],
	extra []executor.Extension[*substate.Substate],
) error {
	var extensions = []executor.Extension[*substate.Substate]{
		tracker.MakeProgressLogger[*substate.Substate](cfg, 0),
		tracker.MakeProgressTracker(cfg, 0),
		statedb.MakeTemporaryStatePrepper(),
		statedb.MakeTemporaryProxyRecorderPrepper(cfg),
	}

	extensions = append(extensions, extra...)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From: int(cfg.First),
			To:   int(cfg.Last) + 1,
		},
		processor,
		extensions,
	)
}

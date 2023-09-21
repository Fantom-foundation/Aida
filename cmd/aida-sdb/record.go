package main

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/action_provider"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	state_db "github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// TraceRecordCommand data structure for the record app
var TraceRecordCommand = cli.Command{
	Action:    Record,
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

func Record(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	// force enable transaction validation
	cfg.ValidateTxState = true

	substate.RecordReplay = true
	substateDb, err := action_provider.OpenSubstateDb(cfg, ctx)
	if err != nil {
		return err
	}
	defer substateDb.Close()

	stateDb, _, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}
	defer stateDb.Close()

	rCtx, err := context.NewRecord(cfg.TraceFile, cfg.First)
	if err != nil {
		return err
	}

	rec := newRecorder(cfg, rCtx)
	if err != nil {
		return err
	}

	defer rec.close()

	return record(cfg, substateDb, stateDb, rec)
}

func newRecorder(config *utils.Config, ctx *context.Record) recorder {
	return recorder{
		config:        config,
		ctx:           ctx,
		curSyncPeriod: config.First / config.SyncPeriodLength,
	}
}

type recorder struct {
	config        *utils.Config
	ctx           *context.Record
	curSyncPeriod uint64
}

func (r recorder) Process(state executor.State, context *executor.Context) error {
	if !r.ctx.Debug {
		r.ctx.Debug = r.config.Debug && (uint64(state.Block) >= r.config.DebugFrom)
	}

	operation.WriteOp(r.ctx, operation.NewEndBlock())

	newSyncPeriod := uint64(state.Block) / r.config.SyncPeriodLength
	for r.curSyncPeriod < newSyncPeriod {
		operation.WriteOp(r.ctx, operation.NewEndSyncPeriod())
		r.curSyncPeriod++
		operation.WriteOp(r.ctx, operation.NewBeginSyncPeriod(r.curSyncPeriod))
	}

	operation.WriteOp(r.ctx, operation.NewBeginBlock(uint64(state.Block)))

	context.State = state_db.MakeInMemoryStateDB(&state.Substate.InputAlloc, uint64(state.Block))
	context.State = proxy.NewRecorderProxy(context.State, r.ctx)

	_, err := utils.ProcessTx(
		context.State,
		r.config,
		uint64(state.Block),
		state.Transaction,
		state.Substate,
	)

	return err
}

func (r recorder) close() {
	// end last block
	operation.WriteOp(r.ctx, operation.NewEndBlock())
	operation.WriteOp(r.ctx, operation.NewEndSyncPeriod())

	r.ctx.Close()
}

func record(config *utils.Config, provider action_provider.ActionProvider, stateDb state_db.StateDB, rec executor.Processor) error {
	return executor.NewExecutor(provider).Run(
		executor.Params{
			From:  int(config.First),
			To:    int(config.Last) + 1,
			State: stateDb,
		},
		rec,
		[]executor.Extension{
			extension.MakeCpuProfiler(config),
			extension.MakeVirtualMachineStatisticsPrinter(config),
			extension.MakeProgressLogger(config, 15*time.Second),
			extension.MakeProgressTracker(config, 100_000),
			extension.MakeStateDbPreparator(),
			extension.MakeTxValidator(config),
			extension.MakeBlockEventEmitter(),
		},
	)
}

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
		config: config,
		ctx:    ctx,
	}
}

type recorder struct {
	config *utils.Config
	ctx    *context.Record
}

func (r recorder) Process(state executor.State, context *executor.Context) error {
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
	r.ctx.Close()
}

// makeProxyRecorderPrepper creates an extension which creates RecorderProxy before each transaction
func makeProxyRecorderPrepper(ctx *context.Record) proxyRecorderPrepper {
	return proxyRecorderPrepper{
		ctx: ctx,
	}
}

type proxyRecorderPrepper struct {
	extension.NilExtension
	ctx *context.Record
}

// PreTransaction creates a RecorderProxy
func (p proxyRecorderPrepper) PreTransaction(_ executor.State, ctx *executor.Context) error {
	ctx.State = proxy.NewRecorderProxy(ctx.State, p.ctx)
	return nil
}

// makeOperationWriter creates an extension which writes block operations
// 1) before transaction is executed
// 2) after whole run
func makeOperationWriter(cfg *utils.Config, ctx *context.Record) operationWriter {
	return operationWriter{
		config:        cfg,
		ctx:           ctx,
		curSyncPeriod: cfg.First / cfg.SyncPeriodLength,
	}
}

type operationWriter struct {
	extension.NilExtension
	config        *utils.Config
	ctx           *context.Record
	curSyncPeriod uint64
}

// PreTransaction writes block operations into record before executing the transaction
func (w operationWriter) PreTransaction(state executor.State, _ *executor.Context) error {
	if !w.ctx.Debug {
		w.ctx.Debug = w.config.Debug && (uint64(state.Block) >= w.config.DebugFrom)
	}

	operation.WriteOp(w.ctx, operation.NewEndBlock())

	newSyncPeriod := uint64(state.Block) / w.config.SyncPeriodLength
	for w.curSyncPeriod < newSyncPeriod {
		operation.WriteOp(w.ctx, operation.NewEndSyncPeriod())
		w.curSyncPeriod++
		operation.WriteOp(w.ctx, operation.NewBeginSyncPeriod(w.curSyncPeriod))
	}

	operation.WriteOp(w.ctx, operation.NewBeginBlock(uint64(state.Block)))
	return nil
}

// PostRun writes final operations into record
func (w operationWriter) PostRun(executor.State, *executor.Context, error) error {
	operation.WriteOp(w.ctx, operation.NewEndBlock())
	operation.WriteOp(w.ctx, operation.NewEndSyncPeriod())

	return nil
}

func record(config *utils.Config, provider action_provider.ActionProvider, stateDb state_db.StateDB, rec recorder) error {
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

			// order needs to be kept accordingly
			// operationWriter > temporaryStatePrepper > proxyRecorderPrepper
			makeOperationWriter(rec.config, rec.ctx),
			extension.MakeTemporaryStatePrepper(),
			makeProxyRecorderPrepper(rec.ctx),

			extension.MakeStateDbPreparator(),
			extension.MakeBlockEventEmitter(),
		},
	)
}

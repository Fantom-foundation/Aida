package main

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// RecordCommand data structure for the record app
var RecordCommand = cli.Command{
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
	substateDb, err := executor.OpenSubstateDb(cfg, ctx)
	if err != nil {
		return err
	}
	defer substateDb.Close()

	rCtx, err := context.NewRecord(cfg.TraceFile, cfg.First)
	if err != nil {
		return err
	}

	rec := newRecorder(cfg, rCtx)
	if err != nil {
		return err
	}

	return record(cfg, substateDb, rec, rCtx, nil)
}

func newRecorder(cfg *utils.Config, ctx *context.Record) *recorder {
	return &recorder{
		cfg: cfg,
		ctx: ctx,
	}
}

type recorder struct {
	cfg *utils.Config
	ctx *context.Record
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

// makeProxyRecorderPrepper creates an extension which creates RecorderProxy before each transaction
func makeProxyRecorderPrepper(ctx *context.Record) *proxyRecorderPrepper {
	return &proxyRecorderPrepper{
		ctx: ctx,
	}
}

type proxyRecorderPrepper struct {
	extension.NilExtension[*substate.Substate]
	ctx *context.Record
}

// PreTransaction creates a RecorderProxy
func (p *proxyRecorderPrepper) PreTransaction(_ executor.State[*substate.Substate], ctx *executor.Context) error {
	ctx.State = proxy.NewRecorderProxy(ctx.State, p.ctx)
	return nil
}

// makeOperationBlockEmitter creates an extension which writes block operations
// 1) before transaction is executed
// 2) after whole run
func makeOperationBlockEmitter(cfg *utils.Config, ctx *context.Record) executor.Extension[*substate.Substate] {
	return &operationBlockEmitter{
		cfg:                cfg,
		ctx:                ctx,
		curSyncPeriod:      cfg.First / cfg.SyncPeriodLength,
		lastProcessedBlock: int(cfg.First),
	}
}

type operationBlockEmitter struct {
	extension.NilExtension[*substate.Substate]
	cfg                *utils.Config
	ctx                *context.Record
	curSyncPeriod      uint64
	lastProcessedBlock int
	first              bool
}

// PreTransaction writes begin block operations into record
func (e *operationBlockEmitter) PreTransaction(state executor.State[*substate.Substate], _ *executor.Context) error {
	if !e.ctx.Debug {
		e.ctx.Debug = e.cfg.Debug && (uint64(state.Block) >= e.cfg.DebugFrom)
	}

	if e.first {
		operation.WriteOp(e.ctx, operation.NewBeginBlock(uint64(state.Block)))
		e.first = false
	}

	// operation writing needs to be kept in PreTransaction because both TemporaryStatePrepper and
	// proxyRecorderPrepper are called in PreTransaction and need to be called before operationBlockEmitter
	if e.lastProcessedBlock != state.Block {
		operation.WriteOp(e.ctx, operation.NewEndBlock())

		newSyncPeriod := uint64(state.Block) / e.cfg.SyncPeriodLength
		for e.curSyncPeriod < newSyncPeriod {
			operation.WriteOp(e.ctx, operation.NewEndSyncPeriod())
			e.curSyncPeriod++
			operation.WriteOp(e.ctx, operation.NewBeginSyncPeriod(e.curSyncPeriod))
		}

		operation.WriteOp(e.ctx, operation.NewBeginBlock(uint64(state.Block)))

		e.lastProcessedBlock = state.Block
	}
	return nil
}

func (e *operationBlockEmitter) PostRun(executor.State[*substate.Substate], *executor.Context, error) error {
	operation.WriteOp(e.ctx, operation.NewEndBlock())
	operation.WriteOp(e.ctx, operation.NewEndSyncPeriod())
	e.ctx.Close()
	return nil
}

func record(cfg *utils.Config, provider executor.Provider[*substate.Substate], processor executor.Processor[*substate.Substate], rCtx *context.Record, extra []executor.Extension[*substate.Substate]) error {
	var extensions = []executor.Extension[*substate.Substate]{
		tracker.MakeProgressLogger[*substate.Substate](cfg, 15*time.Second),
		tracker.MakeProgressTracker(cfg, 100_000),

		// order needs to be kept accordingly
		// operationBlockEmitter > temporaryStatePrepper > proxyRecorderPrepper
		makeOperationBlockEmitter(cfg, rCtx),
		statedb.MakeTemporaryStatePrepper(),
		makeProxyRecorderPrepper(rCtx),
	}

	extensions = append(extensions, extra...)

	return executor.NewExecutor(provider).Run(
		executor.Params{
			From: int(cfg.First),
			To:   int(cfg.Last) + 1,
		},
		processor,
		extensions,
	)
}

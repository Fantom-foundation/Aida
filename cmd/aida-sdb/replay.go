package main

import (
	"fmt"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/action_provider"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	state_db "github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// TraceReplayCommand data structure for the replay app
var TraceReplayCommand = cli.Command{
	Action:    Replay,
	Name:      "replay",
	Usage:     "executes storage trace",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.CarmenSchemaFlag,
		&utils.ChainIDFlag,
		&utils.CpuProfileFlag,
		&utils.QuietFlag,
		&utils.SyncPeriodLengthFlag,
		&utils.KeepDbFlag,
		&utils.MemoryBreakdownFlag,
		&utils.MemoryProfileFlag,
		&utils.RandomSeedFlag,
		&utils.PrimeThresholdFlag,
		&utils.ProfileFlag,
		&utils.ProfileFileFlag,
		&utils.ProfileIntervalFlag,
		&utils.RandomizePrimingFlag,
		&utils.SkipPrimingFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.StateDbSrcFlag,
		&utils.VmImplementation,
		&utils.DbTmpFlag,
		&utils.UpdateBufferSizeFlag,
		&utils.StateDbLoggingFlag,
		&utils.ShadowDb,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,
		&substate.WorkersFlag,
		&utils.TraceFileFlag,
		&utils.TraceDirectoryFlag,
		&utils.TraceDebugFlag,
		&utils.DebugFromFlag,
		&utils.ValidateFlag,
		&utils.ValidateWorldStateFlag,
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The trace replay command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay storage traces.`,
}

func Replay(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	cfg.ProgressLoggerType = utils.OperationType
	cfg.CopySrcDb = true

	if cfg.DbImpl == "memory" {
		return fmt.Errorf("db-impl memory is not supported")
	}

	operations, err := action_provider.OpenOperations(cfg)
	if err != nil {
		return err
	}

	stateDb, _, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}
	defer stateDb.Close()

	rCtx := context.NewReplay()

	rep := newReplayer(cfg, rCtx)

	return replay(cfg, operations, stateDb, rep)
}

func newReplayer(config *utils.Config, ctx *context.Replay) replayer {

	return replayer{
		config:        config,
		ctx:           ctx,
		curSyncPeriod: config.First / config.SyncPeriodLength,
	}
}

type replayer struct {
	config        *utils.Config
	ctx           *context.Replay
	curSyncPeriod uint64
}

func (r replayer) Process(state executor.State, context *executor.Context) error {
	operation.Execute(state.Operation, context.State, r.ctx)
	if r.config.Debug && int(r.config.DebugFrom) >= state.Block {
		operation.Debug(&r.ctx.Context, state.Operation)
	}
	return nil
}

func replay(config *utils.Config, provider action_provider.ActionProvider, stateDb state_db.StateDB, rep replayer) error {
	return executor.NewExecutor(provider).Run(
		executor.Params{
			From:    int(config.First),
			To:      int(config.Last) + 1,
			State:   stateDb,
			RunMode: executor.OperationMode,
		},
		rep,
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

package vm

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// record-replay: vm replay command
var ReplayCommand = cli.Command{
	Action:    replayAction,
	Name:      "replay",
	Usage:     "executes full state transitions and check output consistency",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&substate.WorkersFlag,
		&substate.SkipTransferTxsFlag,
		&substate.SkipCallTxsFlag,
		&substate.SkipCreateTxsFlag,
		&utils.ChainIDFlag,
		&utils.ProfileEVMCallFlag,
		&utils.MicroProfilingFlag,
		&utils.BasicBlockProfilingFlag,
		&utils.ProfilingDbNameFlag,
		&utils.ChannelBufferSizeFlag,
		&utils.VmImplementation,
		&utils.OnlySuccessfulFlag,
		&utils.CpuProfileFlag,
		&utils.StateDbImplementationFlag,
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The aida-vm replay command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay transactions.`,
}

var vm_duration time.Duration

// data collection execution context
type MicroProfilingCollectorContext struct {
	stats  *vm.MicroProfileStatistic
	ctx    context.Context
	cancel context.CancelFunc
	ch     chan struct{}
}

// data collection execution context
type BasicBlockProfilingCollectorContext struct {
	stats  *vm.BasicBlockProfileStatistic
	ctx    context.Context
	cancel context.CancelFunc
	ch     chan struct{}
}

func resetVmDuration() {
	atomic.StoreInt64((*int64)(&vm_duration), 0)
}

func addVmDuration(delta time.Duration) {
	atomic.AddInt64((*int64)(&vm_duration), (int64)(delta))
}

func getVmDuration() time.Duration {
	return time.Duration(atomic.LoadInt64((*int64)(&vm_duration)))
}

// replayTask replays a transaction substate
func replayTask(cfg *utils.Config, block uint64, tx int, recording *substate.Substate, chainID utils.ChainID, log *logging.Logger) error {
	// If requested, skip failed transactions.
	var (
		statedb state.StateDB
		err     error
	)

	switch strings.ToLower(cfg.DbImpl) {
	case "geth":
		statedb, err = state.MakeOffTheChainStateDB(recording.InputAlloc)
		if err != nil {
			return err
		}
	case "geth-memory", "memory":
		statedb = state.MakeInMemoryStateDB(&recording.InputAlloc, block)
	default:
		return fmt.Errorf("unsupported db type: %s", cfg.DbImpl)
	}

	runtime, err := utils.ProcessTx(statedb, cfg, block, tx, recording)
	if err != nil {
		return fmt.Errorf("failed to process block %v, tx %v; %v", block, tx, err)
	}
	addVmDuration(runtime)

	return nil
}

// create new execution context for a data collector
func NewMicroProfilingCollectorContext() *MicroProfilingCollectorContext {
	dcc := new(MicroProfilingCollectorContext)
	dcc.ctx, dcc.cancel = context.WithCancel(context.Background())
	dcc.ch = make(chan struct{})
	dcc.stats = vm.NewMicroProfileStatistic()
	return dcc
}

// create new execution context for a data collector
func NewBasicBlockProfilingCollectorContext() *BasicBlockProfilingCollectorContext {
	dcc := new(BasicBlockProfilingCollectorContext)
	dcc.ctx, dcc.cancel = context.WithCancel(context.Background())
	dcc.ch = make(chan struct{})
	dcc.stats = vm.NewBasicBlockProfileStatistic()
	return dcc
}

// record-replay: func replayAction for replay command
func replayAction(ctx *cli.Context) error {
	var err error

	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	log := logger.NewLogger(cfg.LogLevel, "Substate Replay")

	// spawn contexts for data collector workers
	if cfg.MicroProfiling {
		var dcc [5]*MicroProfilingCollectorContext
		for i := 0; i < 5; i++ {
			dcc[i] = NewMicroProfilingCollectorContext()
			go vm.MicroProfilingCollector(dcc[i].ctx, dcc[i].ch, dcc[i].stats)
		}

		defer func() {
			// cancel collectors
			for i := 0; i < 5; i++ {
				(dcc[i].cancel)() // stop data collector
				<-(dcc[i].ch)     // wait for data collector to finish
			}

			// merge all stats from collectors
			var stats = vm.NewMicroProfileStatistic()
			for i := 0; i < 5; i++ {
				stats.Merge(dcc[i].stats)
			}
			version := fmt.Sprintf("chaind-id:%v", cfg.ChainID)
			stats.Dump(version)
			log.Noticef("aida-vm: replay-action: recorded micro profiling statistics in %v", vm.MicroProfilingDB)
		}()

	}

	if cfg.BasicBlockProfiling {
		var dcc [5]*BasicBlockProfilingCollectorContext
		for i := 0; i < 5; i++ {
			dcc[i] = NewBasicBlockProfilingCollectorContext()
			go vm.BasicBlockProfilingCollector(dcc[i].ctx, dcc[i].ch, dcc[i].stats)
		}

		defer func() {
			// cancel collectors
			for i := 0; i < 5; i++ {
				(dcc[i].cancel)() // stop data collector
				<-(dcc[i].ch)     // wait for data collector to finish
			}

			// merge all stats from collectors
			var stats = vm.NewBasicBlockProfileStatistic()
			for i := 0; i < 5; i++ {
				stats.Merge(dcc[i].stats)
			}
			stats.Dump()
			log.Noticef("recorded basic block profiling statistics in %v\n", vm.BasicBlockProfilingDB)
		}()
	}

	log.Infof("chain-id: %v\n", cfg.ChainID)

	if cfg.ProfileEVMCall {
		evmcore.ProfileEVMCall = true
	}

	if cfg.MicroProfiling {
		vm.MicroProfiling = true
		vm.MicroProfilingBufferSize = cfg.ChannelBufferSize
		vm.MicroProfilingDB = cfg.ProfilingDbName
	}

	if cfg.BasicBlockProfiling {
		vm.BasicBlockProfiling = true
		vm.BasicBlockProfilingBufferSize = cfg.ChannelBufferSize
		vm.BasicBlockProfilingDB = cfg.ProfilingDbName
	}

	substate.SetSubstateDb(cfg.AidaDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	// Start CPU profiling if requested.
	profile_file_name := cfg.CPUProfile
	if profile_file_name != "" {
		f, err := os.Create(profile_file_name)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	task := func(block uint64, tx int, recording *substate.Substate, taskPool *substate.SubstateTaskPool) error {
		return replayTask(cfg, block, tx, recording, cfg.ChainID, log)
	}

	resetVmDuration()
	taskPool := substate.NewSubstateTaskPool("aida-vm replay", task, cfg.First, cfg.Last, ctx)
	err = taskPool.Execute()

	log.Noticef("net VM time: %v\n", getVmDuration())
	utils.PrintEvmStatistics(cfg)

	return err
}

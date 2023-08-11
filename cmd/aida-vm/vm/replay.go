package vm

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"runtime/pprof"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/Tosca/go/vm/lfvm"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
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
		&substate.SubstateDbFlag,
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
		&logger.LogLevelFlag,
	},
	Description: `
The aida-vm replay command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay transactions.`,
}

var vm_duration time.Duration

type ReplayConfig struct {
	vm_impl         string
	only_successful bool
	state_db_impl   string
}

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
func replayTask(config ReplayConfig, block uint64, tx int, recording *substate.Substate, chainID utils.ChainID, log *logging.Logger) error {
	if tx == utils.PseudoTx {
		return nil
	}
	// If requested, skip failed transactions.
	if config.only_successful && recording.Result.Status != types.ReceiptStatusSuccessful {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("Execution of block %d / tx %d paniced: %v", block, tx, r)
		}
	}()

	inputAlloc := recording.InputAlloc
	inputEnv := recording.Env
	inputMessage := recording.Message

	outputAlloc := recording.OutputAlloc
	outputResult := recording.Result

	var (
		vmConfig    vm.Config
		chainConfig *params.ChainConfig
	)

	vmConfig = opera.DefaultVMConfig
	vmConfig.NoBaseFee = true

	chainConfig = utils.GetChainConfig(chainID)

	var hashError error
	getHash := func(num uint64) common.Hash {
		if inputEnv.BlockHashes == nil {
			hashError = fmt.Errorf("getHash(%d) invoked, no blockhashes provided", num)
			return common.Hash{}
		}
		h, ok := inputEnv.BlockHashes[num]
		if !ok {
			hashError = fmt.Errorf("getHash(%d) invoked, blockhash for that block not provided", num)
		}
		return h
	}

	// TODO: implement other state db types
	var statedb state.StateDB
	switch strings.ToLower(config.state_db_impl) {
	case "geth":
		statedb = state.MakeOffTheChainStateDB(inputAlloc)
	case "geth-memory", "memory":
		statedb = state.MakeInMemoryStateDB(&inputAlloc, block)
	default:
		return fmt.Errorf("unsupported db type: %s", config.state_db_impl)
	}

	// Apply Message
	var (
		gaspool   = new(evmcore.GasPool)
		blockHash = common.Hash{0x01}
		txHash    = common.Hash{0x02}
		txIndex   = tx
	)

	gaspool.AddGas(inputEnv.GasLimit)
	blockCtx := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    inputEnv.Coinbase,
		BlockNumber: new(big.Int).SetUint64(inputEnv.Number),
		Time:        new(big.Int).SetUint64(inputEnv.Timestamp),
		Difficulty:  inputEnv.Difficulty,
		GasLimit:    inputEnv.GasLimit,
		GetHash:     getHash,
	}
	// If currentBaseFee is defined, add it to the vmContext.
	if inputEnv.BaseFee != nil {
		blockCtx.BaseFee = new(big.Int).Set(inputEnv.BaseFee)
	}

	msg := inputMessage.AsMessage()

	vmConfig.Tracer = nil
	vmConfig.Debug = false
	vmConfig.InterpreterImpl = config.vm_impl
	statedb.Prepare(txHash, txIndex)

	txCtx := evmcore.NewEVMTxContext(msg)

	evm := vm.NewEVM(blockCtx, txCtx, statedb, chainConfig, vmConfig)

	snapshot := statedb.Snapshot()
	start := time.Now()
	msgResult, err := evmcore.ApplyMessage(evm, msg, gaspool)
	addVmDuration(time.Since(start))

	if err != nil {
		statedb.RevertToSnapshot(snapshot)
		return err
	}

	if hashError != nil {
		return hashError
	}

	if chainConfig.IsByzantium(blockCtx.BlockNumber) {
		statedb.Finalise(true)
	} else {
		statedb.IntermediateRoot(chainConfig.IsEIP158(blockCtx.BlockNumber))
	}

	if err := statedb.Error(); err != nil {
		return err
	}

	evmResult := &substate.SubstateResult{}
	if msgResult.Failed() {
		evmResult.Status = types.ReceiptStatusFailed
	} else {
		evmResult.Status = types.ReceiptStatusSuccessful
	}
	evmResult.Logs = statedb.GetLogs(txHash, blockHash)
	evmResult.Bloom = types.BytesToBloom(types.LogsBloom(evmResult.Logs))
	if to := msg.To(); to == nil {
		evmResult.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, msg.Nonce())
	}
	evmResult.GasUsed = msgResult.UsedGas

	evmAlloc := statedb.GetSubstatePostAlloc()

	r := outputResult.Equal(evmResult)
	a := outputAlloc.Equal(evmAlloc)
	if !(r && a) {
		log.Infof("block: %v Transaction: %v", block, tx)
		if !r {
			log.Criticalf("inconsistent output: result")
			utils.PrintResultDiffSummary(outputResult, evmResult)
		}
		if !a {
			log.Criticalf("inconsistent output: alloc")
			utils.PrintAllocationDiffSummary(&outputAlloc, &evmAlloc)
		}
		return fmt.Errorf("inconsistent output")
	}

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

	substate.SetSubstateDb(cfg.SubstateDb)
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

	var config = ReplayConfig{
		vm_impl:         cfg.VmImpl,
		only_successful: cfg.OnlySuccessful,
		state_db_impl:   cfg.DbImpl,
	}

	task := func(block uint64, tx int, recording *substate.Substate, taskPool *substate.SubstateTaskPool) error {
		return replayTask(config, block, tx, recording, cfg.ChainID, log)
	}

	resetVmDuration()
	taskPool := substate.NewSubstateTaskPool("aida-vm replay", task, cfg.First, cfg.Last, ctx)
	err = taskPool.Execute()

	log.Noticef("net VM time: %v\n", getVmDuration())
	if strings.HasSuffix(cfg.VmImpl, "-stats") {
		lfvm.PrintCollectedInstructionStatistics()
	}

	return err
}

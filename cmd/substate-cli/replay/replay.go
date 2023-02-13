package replay

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"runtime/pprof"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/Aida/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"

	//"github.com/ethereum/go-ethereum/core/state"
	"github.com/Fantom-foundation/Aida/substate-cli/state"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/lfvm"
	_ "github.com/ethereum/go-ethereum/core/vm/lfvm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// record-replay: substate-cli replay command
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
		&substate.SubstateDirFlag,
		&ChainIDFlag,
		&ProfileEVMCallFlag,
		&MicroProfilingFlag,
		&BasicBlockProfilingFlag,
		&DatabaseNameFlag,
		&ChannelBufferSizeFlag,
		&InterpreterImplFlag,
		&OnlySuccessfulFlag,
		&CpuProfilingFlag,
		&UseInMemoryStateDbFlag,
	},
	Description: `
The substate-cli replay command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay transactions.`,
}

var vm_duration time.Duration

type ReplayConfig struct {
	vm_impl          string
	only_successful  bool
	use_in_memory_db bool
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
func replayTask(config ReplayConfig, block uint64, tx int, recording *substate.Substate, taskPool *substate.SubstateTaskPool) error {

	// If requested, skip failed transactions.
	if config.only_successful && recording.Result.Status != types.ReceiptStatusSuccessful {
		return nil
	}

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

	var statedb state.StateDB
	if config.use_in_memory_db {
		statedb = state.MakeInMemoryStateDB(&inputAlloc, block)
	} else {
		statedb = state.MakeOffTheChainStateDB(inputAlloc)
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
		fmt.Printf("block: %v Transaction: %v\n", block, tx)
		if !r {
			fmt.Printf("inconsistent output: result\n")
			PrintResultDiffSummary(outputResult, evmResult)
		}
		if !a {
			fmt.Printf("inconsistent output: alloc\n")
			PrintAllocationDiffSummary(&outputAlloc, &evmAlloc)
		}
		return fmt.Errorf("inconsistent output")
	}

	return nil
}

func printIfDifferent[T comparable](label string, want, have T) bool {
	if want != have {
		fmt.Printf("  Different %s:\n", label)
		fmt.Printf("    want: %v\n", want)
		fmt.Printf("    have: %v\n", have)
		return true
	}
	return false
}

func printIfDifferentBytes(label string, want, have []byte) bool {
	if !bytes.Equal(want, have) {
		fmt.Printf("  Different %s:\n", label)
		fmt.Printf("    want: %v\n", want)
		fmt.Printf("    have: %v\n", have)
		return true
	}
	return false
}

func printIfDifferentBigInt(label string, want, have *big.Int) bool {
	if want == nil && have == nil {
		return false
	}
	if want == nil || have == nil || want.Cmp(have) != 0 {
		fmt.Printf("  Different %s:\n", label)
		fmt.Printf("    want: %v\n", want)
		fmt.Printf("    have: %v\n", have)
		return true
	}
	return false
}

func PrintResultDiffSummary(want, have *substate.SubstateResult) {
	printIfDifferent("status", want.Status, have.Status)
	printIfDifferent("contract address", want.ContractAddress, have.ContractAddress)
	printIfDifferent("gas usage", want.GasUsed, have.GasUsed)
	printIfDifferent("log bloom filter", want.Bloom, have.Bloom)
	if !printIfDifferent("log size", len(want.Logs), len(have.Logs)) {
		for i := range want.Logs {
			printLogDiffSummary(fmt.Sprintf("log[%d]", i), want.Logs[i], have.Logs[i])
		}
	}
}

func printLogDiffSummary(label string, want, have *types.Log) {
	printIfDifferent(fmt.Sprintf("%s.address", label), want.Address, have.Address)
	if !printIfDifferent(fmt.Sprintf("%s.Topics size", label), len(want.Topics), len(have.Topics)) {
		for i := range want.Topics {
			printIfDifferent(fmt.Sprintf("%s.Topics[%d]", label, i), want.Topics[i], have.Topics[i])
		}
	}
	printIfDifferentBytes(fmt.Sprintf("%s.data", label), want.Data, have.Data)
}

func PrintAllocationDiffSummary(want, have *substate.SubstateAlloc) {
	printIfDifferent("substate alloc size", len(*want), len(*have))
	for key := range *want {
		_, present := (*have)[key]
		if !present {
			fmt.Printf("    missing key=%v\n", key)
		}
	}

	for key := range *have {
		_, present := (*want)[key]
		if !present {
			fmt.Printf("    extra key=%v\n", key)
		}
	}

	for key, is := range *have {
		should, present := (*want)[key]
		if present {
			printAccountDiffSummary(fmt.Sprintf("key=%v:", key), should, is)
		}
	}
}

func printAccountDiffSummary(label string, want, have *substate.SubstateAccount) {
	printIfDifferent(fmt.Sprintf("%s.Nonce", label), want.Nonce, have.Nonce)
	printIfDifferentBigInt(fmt.Sprintf("%s.Balance", label), want.Balance, have.Balance)
	printIfDifferentBytes(fmt.Sprintf("%s.Code", label), want.Code, have.Code)

	printIfDifferent(fmt.Sprintf("len(%s.Storage)", label), len(want.Storage), len(have.Storage))
	for key := range want.Storage {
		_, present := have.Storage[key]
		if !present {
			fmt.Printf("    %s.Storage misses key %v\n", label, key)
		}
	}

	for key := range have.Storage {
		_, present := want.Storage[key]
		if !present {
			fmt.Printf("    %s.Storage has extra key %v\n", label, key)
		}
	}

	for key, is := range have.Storage {
		should, present := want.Storage[key]
		if present {
			printIfDifferent(fmt.Sprintf("%s.Storage[%v]", label, key), should, is)
		}
	}
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

	if ctx.Args().Len() != 2 {
		return fmt.Errorf("substate-cli replay command requires exactly 2 arguments")
	}

	// spawn contexts for data collector workers
	if ctx.Bool(MicroProfilingFlag.Name) {
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
			version := fmt.Sprintf("git-date:%v, git-commit:%v, chaind-id:%v", gitDate, gitCommit, chainID)
			stats.Dump(version)
			fmt.Printf("substate-cli replay: recorded micro profiling statistics in %v\n", vm.MicroProfilingDB)
		}()

	}

	if ctx.Bool(BasicBlockProfilingFlag.Name) {
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
			fmt.Printf("substate-cli replay: recorded basic block profiling statistics in %v\n", vm.BasicBlockProfilingDB)
		}()
	}

	chainID = ctx.Int(ChainIDFlag.Name)
	fmt.Printf("chain-id: %v\n", chainID)
	fmt.Printf("git-date: %v\n", gitDate)
	fmt.Printf("git-commit: %v\n", gitCommit)

	first, last, argErr := utils.SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1))
	if argErr != nil {
		return argErr
	}

	if ctx.Bool(ProfileEVMCallFlag.Name) {
		evmcore.ProfileEVMCall = true
	}

	if ctx.Bool(MicroProfilingFlag.Name) {
		vm.MicroProfiling = true
		vm.MicroProfilingBufferSize = ctx.Int(ChannelBufferSizeFlag.Name)
		vm.MicroProfilingDB = ctx.String(DatabaseNameFlag.Name)
	}

	if ctx.Bool(BasicBlockProfilingFlag.Name) {
		vm.BasicBlockProfiling = true
		vm.BasicBlockProfilingBufferSize = ctx.Int(ChannelBufferSizeFlag.Name)
		vm.BasicBlockProfilingDB = ctx.String(DatabaseNameFlag.Name)
	}

	substate.SetSubstateFlags(ctx)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	// Start CPU profiling if requested.
	profile_file_name := ctx.String(CpuProfilingFlag.Name)
	if profile_file_name != "" {
		f, err := os.Create(profile_file_name)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	var config = ReplayConfig{
		vm_impl:          ctx.String(InterpreterImplFlag.Name),
		only_successful:  ctx.Bool(OnlySuccessfulFlag.Name),
		use_in_memory_db: ctx.Bool(UseInMemoryStateDbFlag.Name),
	}

	task := func(block uint64, tx int, recording *substate.Substate, taskPool *substate.SubstateTaskPool) error {
		return replayTask(config, block, tx, recording, taskPool)
	}

	resetVmDuration()
	taskPool := substate.NewSubstateTaskPool("substate-cli replay", task, first, last, ctx)
	err = taskPool.Execute()

	fmt.Printf("substate-cli replay: net VM time: %v\n", getVmDuration())
	if strings.HasSuffix(ctx.String(InterpreterImplFlag.Name), "-stats") {
		lfvm.PrintCollectedInstructionStatistics()
	}

	return err
}

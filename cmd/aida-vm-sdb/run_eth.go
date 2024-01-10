package main

import (
	"errors"
	"fmt"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/urfave/cli/v2"
)

var RunEthTestsCmd = cli.Command{
	Action:    RunEth,
	Name:      "geth",
	Usage:     "Iterates over substates that are executed into a StateDb",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		//// AidaDb
		//&utils.AidaDbFlag,
		//
		//// StateDb
		//&utils.CarmenSchemaFlag,
		//&utils.StateDbImplementationFlag,
		//&utils.StateDbVariantFlag,
		//&utils.StateDbSrcFlag,
		//&utils.DbTmpFlag,
		//&utils.StateDbLoggingFlag,
		//&utils.ValidateStateHashesFlag,
		//
		//// ArchiveDb
		//&utils.ArchiveModeFlag,
		//&utils.ArchiveQueryRateFlag,
		//&utils.ArchiveMaxQueryAgeFlag,
		//&utils.ArchiveVariantFlag,
		//
		//// ShadowDb
		//&utils.ShadowDb,
		//&utils.ShadowDbImplementationFlag,
		//&utils.ShadowDbVariantFlag,
		//
		//// VM
		//&utils.VmImplementation,
		//
		//// Profiling
		//&utils.CpuProfileFlag,
		//&utils.CpuProfilePerIntervalFlag,
		//&utils.DiagnosticServerFlag,
		//&utils.MemoryBreakdownFlag,
		//&utils.MemoryProfileFlag,
		//&utils.RandomSeedFlag,
		//&utils.PrimeThresholdFlag,
		//&utils.ProfileFlag,
		//&utils.ProfileDepthFlag,
		//&utils.ProfileFileFlag,
		//&utils.ProfileSqlite3Flag,
		//&utils.ProfileIntervalFlag,
		//&utils.ProfileDBFlag,
		//&utils.ProfileBlocksFlag,
		//
		//// Priming
		//&utils.RandomizePrimingFlag,
		//&utils.SkipPrimingFlag,
		//&utils.UpdateBufferSizeFlag,
		//
		//// Utils
		&substate.WorkersFlag,
		&utils.ChainIDFlag,
		//&utils.ContinueOnFailureFlag,
		//&utils.SyncPeriodLengthFlag,
		//&utils.KeepDbFlag,
		////&utils.MaxNumTransactionsFlag,
		//&utils.ValidateTxStateFlag,
		//&utils.ValidateFlag,
		//&logger.LogLevelFlag,
		//&utils.NoHeartbeatLoggingFlag,
		//&utils.TrackProgressFlag,
		//&utils.ErrorLoggingFlag,
	},
	Description: `
The aida-vm-sdb substate command requires two arguments: <blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and last block of
the inclusive range of blocks.`,
}

// RunEth performs sequential block processing on a StateDb
func RunEth(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.PathArg)
	if err != nil {
		return err
	}
	//
	//cfg.StateValidationMode = utils.SubsetCheck
	//
	//substateDb, err := executor.OpenSubstateDb(cfg, ctx)
	//if err != nil {
	//	return err
	//}
	//defer substateDb.Close()

	//bt := new(testMatcher)
	//
	//bt.walk()
	//
	//fmt.Println(b)

	return runEth(cfg, executor.NewEthTestProvider(cfg), nil, ethTestProcessor{cfg}, nil)
}

type ethTestProcessor struct {
	cfg *utils.Config
}

func (p ethTestProcessor) Process(state executor.State[*ethtest.Data], ctx *executor.Context) (finalError error) {
	var (
		gaspool = new(core.GasPool)
		txHash  = common.HexToHash(fmt.Sprintf("0x%016d%016d", state.Block, state.Transaction))
		//validate = p.cfg.Validate
	)

	// create vm config
	vmConfig := opera.DefaultVMConfig
	vmConfig.InterpreterImpl = "geth"
	vmConfig.NoBaseFee = true
	vmConfig.Tracer = nil
	vmConfig.Debug = false

	chainConfig := utils.GetChainConfig(p.cfg.ChainID)

	// prepare tx
	gaspool.AddGas(state.Data.Env.GasLimit.Uint64())
	ctx.State.Prepare(txHash, state.Transaction)
	blockCtx := prepareBlockCtx(state.Data.Env)
	txCtx := core.NewEVMTxContext(state.Data.Msg)
	evm := vm.NewEVM(*blockCtx, txCtx, ctx.State, chainConfig, vmConfig)
	snapshot := ctx.State.Snapshot()

	// apply
	_, err := core.ApplyMessage(evm, state.Data.Msg, gaspool)
	if err != nil {
		// if transaction fails, revert to the first snapshot.
		ctx.State.RevertToSnapshot(snapshot)
		finalError = errors.Join(fmt.Errorf("block: %v transaction: %v", state.Block, state.Transaction), err)
		//validate = false
	}

	return
}

func prepareBlockCtx(env *ethtest.Env) *vm.BlockContext {
	getHash := func(_ uint64) common.Hash {
		return env.Hash
	}
	blockCtx := &vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    env.Coinbase,
		BlockNumber: env.Number.Convert(),
		Time:        env.Timestamp.Convert(),
		Difficulty:  env.Difficulty.Convert(),
		GasLimit:    env.GasLimit.Uint64(),
		GetHash:     getHash,
	}
	// If currentBaseFee is defined, add it to the vmContext.
	if env.BaseFee != nil {
		blockCtx.BaseFee = env.BaseFee.Convert()
	}
	return blockCtx
}

func runEth(
	cfg *utils.Config,
	provider executor.Provider[*ethtest.Data],
	stateDb state.StateDB,
	processor executor.Processor[*ethtest.Data],
	extra []executor.Extension[*ethtest.Data],
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[*ethtest.Data]{
		profiler.MakeCpuProfiler[*ethtest.Data](cfg),
		profiler.MakeDiagnosticServer[*ethtest.Data](cfg),
	}

	if stateDb == nil {
		extensionList = append(
			extensionList,
			statedb.NewTemporaryEthStatePrepper(cfg),
			statedb.MakeStateDbManager[*ethtest.Data](cfg),
			statedb.MakeLiveDbBlockChecker[*ethtest.Data](cfg),
			tracker.MakeDbLogger[*ethtest.Data](cfg),
		)
	}

	extensionList = append(extensionList, extra...)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:                   int(cfg.First),
			To:                     int(cfg.Last) + 1,
			NumWorkers:             1, // vm-sdb can run only with one worker
			State:                  stateDb,
			ParallelismGranularity: executor.BlockLevel,
		},
		processor,
		extensionList,
	)
}

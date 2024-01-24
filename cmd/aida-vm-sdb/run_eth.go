package main

import (
	statetest "github.com/Fantom-foundation/Aida/ethtest/state_test"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/logger"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
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
		&utils.CarmenSchemaFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
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
		&utils.VmImplementation,
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

	cfg.StateValidationMode = utils.SubsetCheck
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

	return runEth(cfg, executor.NewEthTestProvider(cfg), nil, executor.MakeLiveDbTxProcessor(cfg), nil)
}

func runEth(
	cfg *utils.Config,
	provider executor.Provider[statetest.Context],
	stateDb state.StateDB,
	processor executor.Processor[statetest.Context],
	extra []executor.Extension[statetest.Context],
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[statetest.Context]{
		profiler.MakeCpuProfiler[statetest.Context](cfg),
		profiler.MakeDiagnosticServer[statetest.Context](cfg),
		logger.MakeProgressLogger[statetest.Context](cfg, 0),
	}

	if stateDb == nil {
		extensionList = append(
			extensionList,
			statedb.MakeStateDbManager[statetest.Context](cfg),
			statedb.NewTemporaryEthStatePrepper(cfg),
			statedb.MakeStateDbManager[statetest.Context](cfg),
			statedb.MakeLiveDbBlockChecker[statetest.Context](cfg),
			logger.MakeDbLogger[statetest.Context](cfg),
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

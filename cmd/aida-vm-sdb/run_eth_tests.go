package main

import (
	"errors"
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/logger"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	log "github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

var RunEthTestsCmd = cli.Command{
	Action:    RunEthereumTests,
	Name:      "ethereum-tests",
	Usage:     "Execute ethereum tests",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Aliases:   []string{"ethtest"},
	Flags: []cli.Flag{
		// StateDb
		&utils.CarmenSchemaFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.DbTmpFlag,
		&utils.StateDbLoggingFlag,

		// ShadowDb
		&utils.ShadowDb,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,

		// VM
		&utils.VmImplementation,

		// Profiling
		&utils.CpuProfileFlag,
		&utils.CpuProfilePerIntervalFlag,
		&utils.DiagnosticServerFlag,
		&utils.MemoryBreakdownFlag,
		&utils.MemoryProfileFlag,
		&utils.RandomSeedFlag,
		&utils.PrimeThresholdFlag,

		// Utils
		&substate.WorkersFlag,
		&utils.ChainIDFlag,
		&utils.ContinueOnFailureFlag,
		&utils.ValidateFlag,
		&utils.ValidateStateHashesFlag,
		&log.LogLevelFlag,
		&utils.ErrorLoggingFlag,
		&utils.EthTestTypeFlag,
	},
	Description: `
The aida-vm-sdb geth-state-tests command requires one argument: <pathToJsonTest or pathToDirWithJsonTests>`,
}

// RunEthereumTests performs sequential block processing on a StateDb
func RunEthereumTests(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.PathArg)
	if err != nil {
		return err
	}

	switch cfg.EthTestType {
	case utils.EthStateTests:
	case utils.EthBlockChainTests:
		return errors.New("the block-chain tests are not yet supported")
	default:
		return fmt.Errorf("please choose correct ethereum test type (--%v)", utils.EthTestTypeFlag.Name)
	}

	cfg.StateValidationMode = utils.SubsetCheck
	cfg.ValidateTxState = true

	return runEth(cfg, executor.NewEthTestProvider(cfg), nil, executor.MakeLiveDbTxProcessor(cfg), nil)
}

func runEth(
	cfg *utils.Config,
	provider executor.Provider[txcontext.TxContext],
	stateDb state.StateDB,
	processor executor.Processor[txcontext.TxContext],
	extra []executor.Extension[txcontext.TxContext],
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[txcontext.TxContext]{
		profiler.MakeCpuProfiler[txcontext.TxContext](cfg),
		profiler.MakeDiagnosticServer[txcontext.TxContext](cfg),
		logger.MakeProgressLogger[txcontext.TxContext](cfg, 0),
		logger.MakeErrorLogger[txcontext.TxContext](cfg),
	}

	if stateDb == nil {
		extensionList = append(
			extensionList,
			statedb.MakeEthTestDbPrepper(cfg),
			statedb.MakeLiveDbBlockChecker[txcontext.TxContext](cfg),
			logger.MakeDbLogger[txcontext.TxContext](cfg),
			statedb.MakeEthStateTestDbPrimer(cfg), // < to be placed after the DbLogger to log priming operations
		)
	}

	extensionList = append(
		extensionList,
		logger.MakeEthStateTestLogger(cfg),
		validator.MakeEthStateTestValidator(cfg),
		validator.MakeEthBlockTestValidator(cfg),
		statedb.MakeBlockEventEmitter[txcontext.TxContext](),
		validator.MakeShadowDbValidator(cfg),
	)

	extensionList = append(extensionList, extra...)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:                   int(cfg.First),
			To:                     int(cfg.Last) + 1,
			NumWorkers:             1,
			State:                  stateDb,
			ParallelismGranularity: executor.BlockLevel,
		},
		processor,
		extensionList,
	)
}
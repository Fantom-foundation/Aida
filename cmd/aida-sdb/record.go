package main

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	log "github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/urfave/cli/v2"
)

// RecordCommand data structure for the record app
var RecordCommand = cli.Command{
	Action:    RecordStateDbTrace,
	Name:      "record",
	Usage:     "captures and records StateDB operations while processing blocks",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.UpdateBufferSizeFlag,
		&utils.CpuProfileFlag,
		&utils.SyncPeriodLengthFlag,
		&utils.WorkersFlag,
		&utils.ChainIDFlag,
		&utils.TraceFileFlag,
		&utils.TraceDebugFlag,
		&utils.DebugFromFlag,
		&utils.AidaDbFlag,
		&log.LogLevelFlag,
	},
	Description: `
The trace record command requires two arguments:
<blockNumFirst> <blockNumLast>
<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to trace transactions.`,
}

func RecordStateDbTrace(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	// force enable transaction validation
	cfg.ValidateTxState = true

	state.EnableRecordReplay()
	aidaDb, err := db.NewReadOnlyBaseDB(cfg.AidaDb)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %w", err)
	}
	defer aidaDb.Close()

	substateIterator := executor.OpenSubstateProvider(cfg, ctx, aidaDb)
	defer substateIterator.Close()

	return record(cfg, substateIterator, executor.MakeLiveDbTxProcessor(cfg), nil)
}

func record(
	cfg *utils.Config,
	provider executor.Provider[txcontext.TxContext],
	processor executor.Processor[txcontext.TxContext],
	extra []executor.Extension[txcontext.TxContext],
) error {
	var extensions = []executor.Extension[txcontext.TxContext]{
		profiler.MakeCpuProfiler[txcontext.TxContext](cfg),
		tracker.MakeBlockProgressTracker(cfg, 0),
		statedb.MakeTemporaryStatePrepper(cfg),
		statedb.MakeProxyRecorderPrepper[txcontext.TxContext](cfg),
		validator.MakeLiveDbValidator(cfg, validator.ValidateTxTarget{WorldState: true, Receipt: true}),
		statedb.MakeTransactionEventEmitter[txcontext.TxContext](),
	}

	extensions = append(extensions, extra...)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:                   int(cfg.First),
			To:                     int(cfg.Last) + 1,
			ParallelismGranularity: executor.TransactionLevel,
		},
		processor,
		extensions,
		nil,
	)
}

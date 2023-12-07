package stochastic

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// RecordCommand data structure for the record app
var RecordCommand = cli.Command{
	Action:    RecordStochastic,
	Name:      "record",
	Usage:     "Record StateDb events while processing blocks",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.CpuProfileFlag,
		&utils.SyncPeriodLengthFlag,
		&utils.OutputFlag,
		&substate.WorkersFlag,
		&utils.ChainIDFlag,
		&utils.AidaDbFlag,
	},
	Description: `
The stochastic record command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block for recording events.`,
}

func RecordStochastic(ctx *cli.Context) error {
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

	return record(cfg, substateDb, nil, executor.MakeLiveDbProcessor(cfg), nil)
}

func record(
	cfg *utils.Config,
	provider executor.Provider[*substate.Substate],
	db state.StateDB,
	processor executor.Processor[*substate.Substate],
	extra []executor.Extension[*substate.Substate],
) error {
	var extensions = []executor.Extension[*substate.Substate]{
		profiler.MakeCpuProfiler[*substate.Substate](cfg),
		tracker.MakeProgressLogger[*substate.Substate](cfg, 0),
		tracker.MakeProgressTracker(cfg, 0),
	}

	if db == nil {
		extensions = append(extensions,
			statedb.MakeTemporaryStatePrepper(cfg),
			statedb.MakeEventProxyPrepper[*substate.Substate](cfg),
		)

	}

	extensions = append(
		extensions,
		validator.MakeLiveDbValidator(cfg),
	)

	extensions = append(extensions, extra...)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:  int(cfg.First),
			To:    int(cfg.Last) + 1,
			State: db,
		},
		processor,
		extensions,
	)
}

package trace

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

func ReplaySubstate(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	substateProvider, err := executor.OpenSubstateDb(cfg, ctx)
	if err != nil {
		return err
	}

	operationProvider, err := executor.OpenOperations(cfg)
	if err != nil {
		return err
	}

	defer substateProvider.Close()

	rCtx := context.NewReplay()

	processor := makeSubstateProcessor(cfg, rCtx, operationProvider)

	var extra = []executor.Extension[*substate.Substate]{
		profiler.MakeReplayProfiler[*substate.Substate](cfg, rCtx),
	}

	return replaySubstate(cfg, substateProvider, processor, nil, extra)
}

func makeSubstateProcessor(cfg *utils.Config, rCtx *context.Replay, operationProvider executor.Provider[[]operation.Operation]) *substateProcessor {
	return &substateProcessor{
		operationProcessor: operationProcessor{cfg, rCtx},
		operationProvider:  operationProvider,
	}
}

type substateProcessor struct {
	operationProcessor
	operationProvider executor.Provider[[]operation.Operation]
}

func (p substateProcessor) Process(state executor.State[*substate.Substate], ctx *executor.Context) error {
	return p.operationProvider.Run(state.Block, state.Block, func(t executor.TransactionInfo[[]operation.Operation]) error {
		p.runTransaction(uint64(state.Block), t.Data, ctx.State)
		return nil
	})
}

func replaySubstate(
	cfg *utils.Config,
	provider executor.Provider[*substate.Substate],
	processor executor.Processor[*substate.Substate],
	stateDb state.StateDB,
	extra []executor.Extension[*substate.Substate],
) error {
	var extensionList = []executor.Extension[*substate.Substate]{
		tracker.MakeProgressLogger[*substate.Substate](cfg, 0),
		profiler.MakeMemoryUsagePrinter[*substate.Substate](cfg),
		profiler.MakeMemoryProfiler[*substate.Substate](cfg),
		validator.MakeTxValidator(cfg),
	}

	if stateDb == nil {
		extensionList = append(extensionList, statedb.MakeStateDbManager[*substate.Substate](cfg))
	}

	if cfg.DbImpl == "memory" {
		extensionList = append(extensionList, statedb.MakeStateDbPrepper())
	} else {
		extensionList = append(extensionList, statedb.MakeTxPrimer(cfg))
	}

	extensionList = append(extensionList, extra...)

	return executor.NewExecutor(provider).Run(
		executor.Params{
			From:  int(cfg.First),
			To:    int(cfg.Last) + 1,
			State: stateDb,
		},
		processor,
		extensionList,
	)
}

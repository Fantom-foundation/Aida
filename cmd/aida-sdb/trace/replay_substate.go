package trace

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/primer"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/executor/transaction"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
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

	var extra = []executor.Extension[transaction.SubstateData]{
		profiler.MakeReplayProfiler[transaction.SubstateData](cfg, rCtx),
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

func (p substateProcessor) Process(state executor.State[transaction.SubstateData], ctx *executor.Context) error {
	return p.operationProvider.Run(state.Block, state.Block, func(t executor.TransactionInfo[[]operation.Operation]) error {
		p.runTransaction(uint64(state.Block), t.Data, ctx.State)
		return nil
	})
}

func replaySubstate(
	cfg *utils.Config,
	provider executor.Provider[transaction.SubstateData],
	processor executor.Processor[transaction.SubstateData],
	stateDb state.StateDB,
	extra []executor.Extension[transaction.SubstateData],
) error {
	var extensionList = []executor.Extension[transaction.SubstateData]{
		profiler.MakeCpuProfiler[transaction.SubstateData](cfg),
		tracker.MakeProgressLogger[transaction.SubstateData](cfg, 0),
		profiler.MakeMemoryUsagePrinter[transaction.SubstateData](cfg),
		profiler.MakeMemoryProfiler[transaction.SubstateData](cfg),
		validator.MakeLiveDbValidator(cfg),
	}

	if stateDb == nil {
		extensionList = append(extensionList, statedb.MakeStateDbManager[transaction.SubstateData](cfg))
	}

	if cfg.DbImpl == "memory" {
		extensionList = append(extensionList, statedb.MakeStateDbPrepper())
	} else {
		extensionList = append(extensionList, primer.MakeTxPrimer(cfg))
	}

	extensionList = append(extensionList, extra...)

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:  int(cfg.First),
			To:    int(cfg.Last) + 1,
			State: stateDb,
		},
		processor,
		extensionList,
	)
}

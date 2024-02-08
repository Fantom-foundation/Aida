package main

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/urfave/cli/v2"
)

// RunTxGenerator performs sequential block processing on a StateDb using transaction generator
func RunTxGenerator(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	cfg.StateValidationMode = utils.SubsetCheck

	// todo init the provider (the generator) here and pass it to runTransactions

	return runTransactions(cfg, nil, nil, false)
}
func newGenerateData() txcontext.TxContext {
	return &generateData{}
}

type generateData struct {
	txcontext.NilTxContext
}

func (g generateData) GetStateHash() common.Hash {
	// ignored
	return common.Hash{}
}

func (g generateData) GetOutputState() txcontext.WorldState {
	//TODO implement me
	panic("implement me")
}

func (g generateData) GetBlockEnvironment() txcontext.BlockEnvironment {
	//TODO implement me
	panic("implement me")
}

func (g generateData) GetMessage() core.Message {
	//TODO implement me
	panic("implement me")
}

type txProcessor struct {
	cfg *utils.Config
}

func (p txProcessor) Process(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	// todo apply data onto StateDb
	return nil
}

func runTransactions(
	cfg *utils.Config,
	provider executor.Provider[txcontext.TxContext],
	stateDb state.StateDB,
	disableStateDbExtension bool,
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[txcontext.TxContext]{
		// todo choose extensions
	}

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:  int(cfg.First),
			To:    int(cfg.Last) + 1,
			State: stateDb,
		},
		txProcessor{cfg},
		extensionList,
	)
}

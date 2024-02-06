package statedb

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

func MakeEthStateTestDbPrepper(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	return makeEthStateTestDbPrepper(logger.NewLogger(cfg.LogLevel, "EthStatePrepper"), cfg)
}

func makeEthStateTestDbPrepper(log logger.Logger, cfg *utils.Config) *ethStateTestDbPrepper {
	return &ethStateTestDbPrepper{
		cfg: cfg,
		log: log,
	}
}

type ethStateTestDbPrepper struct {
	extension.NilExtension[txcontext.TxContext]
	cfg *utils.Config
	log logger.Logger
}

func (e *ethStateTestDbPrepper) PreTransaction(st executor.State[txcontext.TxContext], ctx *executor.Context) error {
	err := os.RemoveAll(ctx.StateDbPath)
	if err != nil {
		return fmt.Errorf("cannot remove db %v; %v", ctx.StateDbPath, err)
	}

	ctx.State, ctx.StateDbPath, err = utils.PrepareStateDB(e.cfg)
	if err != nil {
		return fmt.Errorf("failed to prepare statedb; %v", err)
	}

	primeCtx := utils.NewPrimeContext(e.cfg, ctx.State, e.log)

	err = primeCtx.PrimeStateDB(st.Data.GetInputState(), ctx.State)
	if err != nil {
		return err
	}

	return nil
}

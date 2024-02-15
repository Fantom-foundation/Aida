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

func MakeEthTestDbPrepper(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	return makeEthTestDbPrepper(logger.NewLogger(cfg.LogLevel, "EthStatePrepper"), cfg)
}

func makeEthTestDbPrepper(log logger.Logger, cfg *utils.Config) *ethTestDbPrepper {
	return &ethTestDbPrepper{
		cfg: cfg,
		log: log,
	}
}

type ethTestDbPrepper struct {
	extension.NilExtension[txcontext.TxContext]
	cfg *utils.Config
	log logger.Logger
}

func (e ethTestDbPrepper) PreBlock(st executor.State[txcontext.TxContext], ctx *executor.Context) error {
	var err error
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

func (e ethTestDbPrepper) PostBlock(_ executor.State[txcontext.TxContext], ctx *executor.Context) error {
	if ctx.State != nil {
		err := ctx.State.Close()
		if err != nil {
			return fmt.Errorf("cannot close db %v; %v", ctx.StateDbPath, err)
		}
		ctx.State = nil
	}

	err := os.RemoveAll(ctx.StateDbPath)
	if err != nil {
		return fmt.Errorf("cannot remove db %v; %v", ctx.StateDbPath, err)
	}

	return nil
}

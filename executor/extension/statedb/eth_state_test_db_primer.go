package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

func MakeEthStateTestDbPrimer(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	return makeEthStateTestDbPrimer(logger.NewLogger(cfg.LogLevel, "EthStatePrimer"), cfg)
}

func makeEthStateTestDbPrimer(log logger.Logger, cfg *utils.Config) *ethStateTestDbPrimer {
	return &ethStateTestDbPrimer{
		cfg: cfg,
		log: log,
	}
}

type ethStateTestDbPrimer struct {
	extension.NilExtension[txcontext.TxContext]
	cfg *utils.Config
	log logger.Logger
}

func (e ethStateTestDbPrimer) PreBlock(st executor.State[txcontext.TxContext], ctx *executor.Context) error {
	primeCtx := utils.NewPrimeContext(e.cfg, ctx.State, e.log)
	return primeCtx.PrimeStateDB(st.Data.GetInputState(), ctx.State)
}

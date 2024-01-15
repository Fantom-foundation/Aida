package primer

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/executor/transaction"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeTxPrimer creates an extension that primes StateDb before each transaction
func MakeTxPrimer(cfg *utils.Config) executor.Extension[transaction.SubstateData] {
	return makeTxPrimer(cfg, logger.NewLogger(cfg.LogLevel, "TxPrimer"))
}

func makeTxPrimer(cfg *utils.Config, log logger.Logger) executor.Extension[transaction.SubstateData] {
	return &txPrimer{cfg: cfg, log: log}
}

type txPrimer struct {
	extension.NilExtension[transaction.SubstateData]
	primeCtx *utils.PrimeContext
	cfg      *utils.Config
	log      logger.Logger
}

func (p *txPrimer) PreRun(_ executor.State[transaction.SubstateData], ctx *executor.Context) error {
	p.primeCtx = utils.NewPrimeContext(p.cfg, ctx.State, p.log)
	return nil
}

// PreTransaction primes StateDb
func (p *txPrimer) PreTransaction(state executor.State[transaction.SubstateData], ctx *executor.Context) error {
	return p.primeCtx.PrimeStateDB(state.Data.GetInputAlloc(), ctx.State)
}

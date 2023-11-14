package primer

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
)

// MakeTxPrimer creates an extension that primes StateDb before each transaction
func MakeTxPrimer(cfg *utils.Config) executor.Extension[*substate.Substate] {
	return makeTxPrimer(cfg, logger.NewLogger(cfg.LogLevel, "TxPrimer"))
}

func makeTxPrimer(cfg *utils.Config, log logger.Logger) executor.Extension[*substate.Substate] {
	return &txPrimer{cfg: cfg, log: log}
}

type txPrimer struct {
	extension.NilExtension[*substate.Substate]
	primeCtx *primeContext
	cfg      *utils.Config
	log      logger.Logger
}

func (p *txPrimer) PreRun(_ executor.State[*substate.Substate], ctx *executor.Context) error {
	p.primeCtx = newPrimeContext(p.cfg, ctx.State, p.log)
	return nil
}

// PreTransaction primes StateDb
func (p *txPrimer) PreTransaction(state executor.State[*substate.Substate], ctx *executor.Context) error {
	return p.primeCtx.PrimeStateDB(state.Data.InputAlloc, ctx.State)
}

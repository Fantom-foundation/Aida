package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

func MakeStateDbPrimer(config *utils.Config) executor.Extension {
	if config.SkipPriming {
		return NilExtension{}
	}

	return makeStateDbPrimer(config, logger.NewLogger(config.LogLevel, "StateDb-Primer"))
}

func makeStateDbPrimer(config *utils.Config, log logger.Logger) *stateDbPrimer {
	return &stateDbPrimer{
		config: config,
		log:    log,
	}
}

type stateDbPrimer struct {
	NilExtension
	config *utils.Config
	log    logger.Logger
}

// PreRun primes StateDb to given block.
func (p *stateDbPrimer) PreRun(state executor.State) error {
	if p.config.StateDbSrc != "" {
		p.log.Warning("Skipping priming due to usage of preexisting StateDb")
		return nil
	}

	p.log.Noticef("Priming to block %v", p.config.First-1)
	if err := utils.LoadWorldStateAndPrime(state.State, p.config, p.config.First-1); err != nil {
		return err
	}

	return nil
}

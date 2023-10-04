package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

func MakeStateDbPrimer[T any](config *utils.Config) executor.Extension[T] {
	if config.SkipPriming {
		return NilExtension[T]{}
	}

	return makeStateDbPrimer[T](config, logger.NewLogger(config.LogLevel, "StateDb-Primer"))
}

func makeStateDbPrimer[T any](config *utils.Config, log logger.Logger) *stateDbPrimer[T] {
	return &stateDbPrimer[T]{
		config: config,
		log:    log,
	}
}

type stateDbPrimer[T any] struct {
	NilExtension[T]
	config *utils.Config
	log    logger.Logger
}

// PreRun primes StateDb to given block.
func (p *stateDbPrimer[T]) PreRun(state executor.State[T], context *executor.Context) error {
	if p.config.IsExistingStateDb {
		p.log.Warning("Skipping priming due to usage of preexisting StateDb")
		return nil
	}

	p.log.Noticef("Priming to block %v", p.config.First-1)
	if err := utils.LoadWorldStateAndPrime(context.State, p.config, p.config.First-1); err != nil {
		return err
	}

	return nil
}

package statedb

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

func MakeStateDbPrimer[T any](cfg *utils.Config) executor.Extension[T] {
	if cfg.SkipPriming {
		return extension.NilExtension[T]{}
	}

	return makeStateDbPrimer[T](cfg, logger.NewLogger(cfg.LogLevel, "StateDb-Primer"))
}

func makeStateDbPrimer[T any](cfg *utils.Config, log logger.Logger) *stateDbPrimer[T] {
	return &stateDbPrimer[T]{
		cfg: cfg,
		log: log,
	}
}

type stateDbPrimer[T any] struct {
	extension.NilExtension[T]
	cfg *utils.Config
	log logger.Logger
}

// PreRun primes StateDb to given block.
func (p *stateDbPrimer[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	if p.cfg.IsExistingStateDb {
		p.log.Warning("Skipping priming due to usage of preexisting StateDb")
		return nil
	}

	if p.cfg.First == 0 {
		return nil
	}

	p.log.Noticef("Priming to block %v", p.cfg.First-1)
	err := utils.LoadWorldStateAndPrime(ctx.State, p.cfg, p.cfg.First-1)
	if err != nil {
		return fmt.Errorf("cannot prime; %v", err)
	}

	return nil
}

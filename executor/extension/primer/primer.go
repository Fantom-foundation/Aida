package primer

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/google/martian/log"
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
	ctx *utils.PrimeContext
}

// PreRun primes StateDb to given block.
func (p *stateDbPrimer[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	if p.cfg.IsExistingStateDb {
		p.log.Warning("Skipping priming due to usage of pre-existing StateDb")
		return nil
	}

	if p.cfg.SkipPriming {
		p.log.Warning("Skipping priming (disabled by user)...")
		return nil
	}

	if p.cfg.First == 0 {
		return nil
	}

	p.log.Noticef("Priming to block %v", p.cfg.First-1)
	p.ctx = utils.NewPrimeContext(p.cfg, ctx.State, p.log)

	return p.prime(ctx.State)
}

// prime the stateDb to given first block.
func (p *stateDbPrimer[T]) prime(stateDb state.StateDB) error {
	var (
		totalSize uint64 // total size of unprimed update set
		block     uint64 // current block position
		hasPrimed bool   // if true, db has been primed
	)

	// load pre-computed update-set from update-set db
	udb, err := substate.OpenUpdateDBReadOnly(p.cfg.UpdateDb)
	if err != nil {
		return err
	}

	defer udb.Close()
	updateIter := substate.NewUpdateSetIterator(udb, block, p.cfg.First-1)
	update := make(substate.SubstateAlloc)

	for updateIter.Next() {
		newSet := updateIter.Value()
		if newSet.Block > p.cfg.First-1 {
			break
		}
		block = newSet.Block

		incrementalSize := update.EstimateIncrementalSize(*newSet.UpdateSet)

		// Prime StateDB
		if totalSize+incrementalSize > p.cfg.UpdateBufferSize {
			p.log.Infof("\tPriming...")
			if err = p.ctx.PrimeStateDB(update, stateDb); err != nil {
				return fmt.Errorf("cannot prime state-db; %v", err)
			}

			totalSize = 0
			update = make(substate.SubstateAlloc)
			hasPrimed = true
		}

		// Reset accessed storage locations of suicided accounts prior to update-set block.
		// The known accessed storage locations in the update-set range has already been
		// reset when generating the update set database.
		utils.ClearAccountStorage(update, newSet.DeletedAccounts)
		// if exists in DB, suicide
		if hasPrimed {
			p.ctx.SuicideAccounts(stateDb, newSet.DeletedAccounts)
		}

		update.Merge(*newSet.UpdateSet)
		totalSize += incrementalSize
		p.log.Infof("\tMerge update set at block %v. New total size %v MB (+%v MB)",
			newSet.Block, totalSize/1_000_000,
			incrementalSize/1_000_000)
		// advance block after merge update set
		block++
	}

	// if update set is not empty, prime the remaining
	if len(update) > 0 {
		if err = p.ctx.PrimeStateDB(update, stateDb); err != nil {
			return fmt.Errorf("cannot prime state-db; %v", err)
		}
		update = make(substate.SubstateAlloc)
		hasPrimed = true
	}
	updateIter.Release()

	// advance from the latest precomputed update-set to the target block
	// if the first block is 1, target must prime the genesis block
	if block < p.cfg.First-1 || p.cfg.First-1 == 0 {
		log.Infof("\tPriming using substate from %v to %v", block, p.cfg.First-1)
		update, deletedAccounts, err := utils.GenerateUpdateSet(block, p.cfg.First-1, p.cfg)
		if err != nil {
			return fmt.Errorf("cannot generate update-set; %v", err)
		}
		if hasPrimed {
			p.ctx.SuicideAccounts(stateDb, deletedAccounts)
		}
		if err = p.ctx.PrimeStateDB(update, stateDb); err != nil {
			return fmt.Errorf("cannot prime state-db; %v", err)
		}
	}

	p.log.Noticef("Delete destroyed accounts until block %v", p.cfg.First-1)

	// remove destroyed accounts until one block before the first block
	err = utils.DeleteDestroyedAccountsFromStateDB(stateDb, p.cfg, p.cfg.First-1)
	if err != nil {
		return fmt.Errorf("cannot delete destroyed accounts from state-db; %v", err)
	}

	return nil
}

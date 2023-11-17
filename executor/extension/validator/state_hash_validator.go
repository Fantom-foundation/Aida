package validator

import (
	"errors"
	"fmt"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb"
)

func MakeStateHashValidator[T any](cfg *utils.Config) executor.Extension[T] {
	if !cfg.ValidateStateHashes {
		return extension.NilExtension[T]{}
	}

	log := logger.NewLogger("INFO", "state-hash-validator")
	return makeStateHashValidator[T](cfg, log)
}

func makeStateHashValidator[T any](cfg *utils.Config, log logger.Logger) *stateHashValidator[T] {
	return &stateHashValidator[T]{cfg: cfg, log: log, nextArchiveBlockToCheck: int(cfg.First)}
}

type stateHashValidator[T any] struct {
	extension.NilExtension[T]
	cfg                     *utils.Config
	log                     logger.Logger
	nextArchiveBlockToCheck int
	lastProcessedBlock      int
	hashProvider            utils.StateHashProvider
	numberOfErrors          int
}

func (e *stateHashValidator[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	if e.cfg.DbImpl == "carmen" {
		if e.cfg.CarmenSchema != 5 {
			return errors.New("state-hash-validation only works with carmen schema 5")
		}

		if e.cfg.ArchiveMode && e.cfg.ArchiveVariant != "s5" {
			return errors.New("archive state-hash-validation only works with archive variant s5")
		}
	} else if e.cfg.DbImpl != "geth" {
		return errors.New("state-hash-validation only works with db-impl carmen or geth")
	}

	e.hashProvider = utils.MakeStateHashProvider(ctx.AidaDb)
	return nil
}

func (e *stateHashValidator[T]) PostBlock(state executor.State[T], ctx *executor.Context) error {
	if ctx.State == nil {
		return nil
	}

	want, err := e.getStateHash(state.Block)
	if err != nil {
		return err
	}

	got := ctx.State.GetHash()
	if want != got {
		err = fmt.Errorf("unexpected hash for Live block %d\nwanted %v\n   got %v", state.Block, want, got)
		if e.isErrorFatal(err, ctx.ErrorInput) {
			return err
		}
	}

	// Check the ArchiveDB
	if e.cfg.ArchiveMode {
		e.lastProcessedBlock = state.Block
		if err = e.checkArchiveHashes(ctx.State); err != nil {
			if e.isErrorFatal(err, ctx.ErrorInput) {
				return err
			}
		}
	}

	return nil
}

func (e *stateHashValidator[T]) PostRun(_ executor.State[T], ctx *executor.Context, err error) error {
	// Skip processing if run is aborted due to an error.
	if err != nil {
		return nil
	}
	// Complete processing remaining archive blocks.
	if e.cfg.ArchiveMode {
		for e.nextArchiveBlockToCheck < e.lastProcessedBlock {
			if err = e.checkArchiveHashes(ctx.State); err != nil {
				ctx.ErrorInput <- err
				return err
			}
			if e.nextArchiveBlockToCheck < e.lastProcessedBlock {
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
	return nil
}

func (e *stateHashValidator[T]) checkArchiveHashes(state state.StateDB) error {
	// Note: the archive may be lagging behind the life DB, so block hashes need
	// to be checked as they become available.
	height, empty, err := state.GetArchiveBlockHeight()
	if err != nil {
		return fmt.Errorf("failed to get archive block height: %v", err)
	}

	cur := uint64(e.nextArchiveBlockToCheck)
	for !empty && cur <= height {

		want, err := e.getStateHash(int(cur))
		if err != nil {
			return err
		}

		archive, err := state.GetArchiveState(cur)
		if err != nil {
			return err
		}

		got := archive.GetHash()
		archive.Release()
		if want != got {
			return fmt.Errorf("unexpected hash for archive block %d\nwanted %v\n   got %v", cur, want, got)
		}

		cur++
	}
	e.nextArchiveBlockToCheck = int(cur)
	return nil
}

func (e *stateHashValidator[T]) getStateHash(blockNumber int) (common.Hash, error) {
	want, err := e.hashProvider.GetStateHash(blockNumber)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return common.Hash{}, fmt.Errorf("state hash for block %v is not present in the db", blockNumber)
		}
		return common.Hash{}, fmt.Errorf("cannot get state hash for block %v; %v", blockNumber, err)
	}

	return want, nil

}

func (e *stateHashValidator[T]) isErrorFatal(err error, ch chan error) bool {
	if !e.cfg.ContinueOnFailure {
		return true
	}

	ch <- err
	e.numberOfErrors++

	// endless run
	if e.cfg.MaxNumErrors == 0 {
		return false
	}

	// too many errors
	if e.numberOfErrors >= e.cfg.MaxNumErrors {
		return true
	}

	return false
}

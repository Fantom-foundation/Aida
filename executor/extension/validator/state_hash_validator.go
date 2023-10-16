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

func MakeStateHashValidator[T any](config *utils.Config) executor.Extension[T] {
	if !config.ValidateStateHashes {
		return extension.NilExtension[T]{}
	}

	log := logger.NewLogger("INFO", "state-hash-validator")
	return makeStateHashValidator[T](config, log)
}

func makeStateHashValidator[T any](config *utils.Config, log logger.Logger) *stateHashValidator[T] {
	return &stateHashValidator[T]{config: config, log: log}
}

type stateHashValidator[T any] struct {
	extension.NilExtension[T]
	config                  *utils.Config
	log                     logger.Logger
	nextArchiveBlockToCheck int
	lastProcessedBlock      int
	hashProvider            utils.StateHashProvider
}

func (e *stateHashValidator[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	e.hashProvider = utils.MakeStateHashProvider(ctx.AidaDb)
	return nil
}

func (e *stateHashValidator[T]) PostBlock(state executor.State[T], context *executor.Context) error {
	if context.State == nil {
		return nil
	}

	want, err := e.getStateHash(state.Block)
	if err != nil {
		return err
	}

	// is stat hash present?
	nilHash := common.Hash{}
	if want == nilHash {
		return nil
	}

	got := context.State.GetHash()
	if want != got {
		return fmt.Errorf("unexpected hash for Live block %d\nwanted %v\n   got %v", state.Block, want, got)
	}

	// Check the ArchiveDB
	if e.config.ArchiveMode {
		e.lastProcessedBlock = state.Block
		if err = e.checkArchiveHashes(context.State); err != nil {
			return err
		}
	}

	return nil
}

func (e *stateHashValidator[T]) PostRun(state executor.State[T], context *executor.Context, err error) error {
	// Skip processing if run is aborted due to an error.
	if err != nil {
		return nil
	}
	// Complete processing remaining archive blocks.
	if e.config.ArchiveMode {
		for e.nextArchiveBlockToCheck < e.lastProcessedBlock {
			if err = e.checkArchiveHashes(context.State); err != nil {
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

		// is state hash present?
		nilHash := common.Hash{}
		if want == nilHash {
			return nil
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
			e.log.Warningf("State hash for block %v is not present in the db", blockNumber)
			return common.Hash{}, nil
		}
		return common.Hash{}, fmt.Errorf("cannot get state hash for block %v; %v", blockNumber, err)
	}

	return want, nil

}

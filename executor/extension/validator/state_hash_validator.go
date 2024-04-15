// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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

	// NOTE: ContinueOnFailure does not make sense here, if hash does not
	// match every block after this block would have different hash
	got, err := ctx.State.GetHash()
	if err != nil {
		return fmt.Errorf("cannot get state hash; %w", err)
	}
	if want != got {
		return fmt.Errorf("unexpected hash for Live block %d\nwanted %v\n   got %v", state.Block, want, got)
	}

	// Check the ArchiveDB
	if e.cfg.ArchiveMode {
		e.lastProcessedBlock = state.Block
		if err = e.checkArchiveHashes(ctx.State); err != nil {
			return err
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

		// NOTE: ContinueOnFailure does not make sense here, if hash does not
		// match every block after this block would have different hash
		got, err := archive.GetHash()
		archive.Release()
		if err != nil {
			return fmt.Errorf("cannot GetHash; %w", err)
		}
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

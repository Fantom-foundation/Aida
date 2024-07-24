// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package statedb

import (
	"fmt"
	"path/filepath"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	umath "github.com/Fantom-foundation/Aida/utils/math"
)

type archiveBlockChecker[T any] struct {
	extension.NilExtension[T]
	cfg *utils.Config
	log logger.Logger
}

// MakeArchiveBlockChecker creates an executor.Extension which checks if given
// archive has archive states block alignment of given Archive StateDb
func MakeArchiveBlockChecker[T any](cfg *utils.Config) executor.Extension[T] {
	return &archiveBlockChecker[T]{
		cfg: cfg,
		log: logger.NewLogger(cfg.LogLevel, "Archive-Block-Checker"),
	}
}

// PreRun checks whether given block range is within given ArchiveDb
func (c *archiveBlockChecker[T]) PreRun(executor.State[T], *executor.Context) error {
	var archiveLastBlock uint64

	if c.cfg.ShadowDb {
		primeDbInfo, err := utils.ReadStateDbInfo(filepath.Join(c.cfg.StateDbSrc, utils.PathToPrimaryStateDb, utils.PathToDbInfo))
		if err != nil {
			return fmt.Errorf("cannot read state db info for primary db; %v", err)
		}

		if !primeDbInfo.ArchiveMode {
			return fmt.Errorf("prime state db %v does not contain archive", filepath.Join(c.cfg.StateDbSrc, utils.PathToPrimaryStateDb))
		}

		shadowDbInfo, err := utils.ReadStateDbInfo(filepath.Join(c.cfg.StateDbSrc, utils.PathToShadowStateDb, utils.PathToDbInfo))
		if err != nil {
			return fmt.Errorf("cannot read state db info for shadow db; %v", err)
		}

		if !shadowDbInfo.ArchiveMode {
			return fmt.Errorf("shadow state db %v does not contain archive", filepath.Join(c.cfg.StateDbSrc, utils.PathToShadowStateDb))
		}

		archiveLastBlock = umath.Min(shadowDbInfo.Block, primeDbInfo.Block)

	} else {
		stateDbInfo, err := utils.ReadStateDbInfo(filepath.Join(c.cfg.StateDbSrc, utils.PathToDbInfo))
		if err != nil {
			return fmt.Errorf("cannot read state db info; %v", err)
		}

		if !stateDbInfo.ArchiveMode {
			return fmt.Errorf("state db %v does not contain archive", c.cfg.StateDbSrc)
		}
		archiveLastBlock = stateDbInfo.Block
	}

	if c.cfg.Last > archiveLastBlock {
		c.log.Noticef("Last block of given archive-db (%v) is smaller than given last block (%v). Adjust last block to %v", archiveLastBlock, c.cfg.Last, archiveLastBlock)
		c.cfg.Last = archiveLastBlock
	}

	return nil
}

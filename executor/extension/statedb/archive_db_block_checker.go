package statedb

import (
	"fmt"
	"path/filepath"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/utils"
	umath "github.com/Fantom-foundation/Aida/utils/math"
)

type archiveBlockChecker[T any] struct {
	extension.NilExtension[T]
	cfg *utils.Config
}

// MakeArchiveBlockChecker creates an executor.Extension which checks if given
// archive has archive states block alignment of given Archive StateDb
func MakeArchiveBlockChecker[T any](cfg *utils.Config) executor.Extension[T] {
	return &archiveBlockChecker[T]{
		cfg: cfg,
	}
}

// PreRun checks whether given block range is within given ArchiveDb
func (c *archiveBlockChecker[T]) PreRun(executor.State[T], *executor.Context) error {
	var lastBlock uint64

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

		lastBlock = umath.Min(shadowDbInfo.Block, primeDbInfo.Block)

	} else {
		stateDbInfo, err := utils.ReadStateDbInfo(filepath.Join(c.cfg.StateDbSrc, utils.PathToDbInfo))
		if err != nil {
			return fmt.Errorf("cannot read state db info; %v", err)
		}

		if !stateDbInfo.ArchiveMode {
			return fmt.Errorf("state db %v does not contain archive", c.cfg.StateDbSrc)
		}
		lastBlock = stateDbInfo.Block
	}

	if c.cfg.Last > lastBlock {
		return fmt.Errorf("last block of given archive-db (%v) is smaller than given last block (%v), please choose a block in range", lastBlock, c.cfg.Last)
	}

	return nil
}

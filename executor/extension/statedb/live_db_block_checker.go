package statedb

import (
	"fmt"
	"path/filepath"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/utils"
)

type liveDbBlockChecker[T any] struct {
	extension.NilExtension[T]
	cfg *utils.Config
}

// MakeLiveDbBlockChecker creates an executor.Extension which checks block alignment of given Live StateDb
func MakeLiveDbBlockChecker[T any](cfg *utils.Config) executor.Extension[T] {
	// this extension is only necessary for existing LiveDb
	if cfg.StateDbSrc == "" {
		return extension.NilExtension[T]{}
	}

	return &liveDbBlockChecker[T]{
		cfg: cfg,
	}
}

// PreRun checks existing LiveDbs block alignment
func (c *liveDbBlockChecker[T]) PreRun(executor.State[T], *executor.Context) error {
	var (
		primeDbInfo  utils.StateDbInfo
		shadowDbInfo utils.StateDbInfo
		lastBlock    uint64
		err          error
	)

	if c.cfg.ShadowDb {
		primeDbInfo, err = utils.ReadStateDbInfo(filepath.Join(c.cfg.StateDbSrc, utils.PathToPrimaryStateDb, utils.PathToDbInfo))
		if err != nil {
			return fmt.Errorf("cannot read state db info for primary db; %v", err)
		}

		shadowDbInfo, err = utils.ReadStateDbInfo(filepath.Join(c.cfg.StateDbSrc, utils.PathToShadowStateDb, utils.PathToDbInfo))
		if err != nil {
			return fmt.Errorf("cannot read state db info for shadow db; %v", err)
		}

		// both shadow and prime dbs have to contain same last block
		if shadowDbInfo.Block != primeDbInfo.Block {
			return fmt.Errorf("shadow (%v) and prime (%v) state dbs have different last block", primeDbInfo.Block, shadowDbInfo.Block)
		}

		lastBlock = primeDbInfo.Block

	} else {
		primeDbInfo, err = utils.ReadStateDbInfo(filepath.Join(c.cfg.StateDbSrc, utils.PathToDbInfo))
		if err != nil {
			return fmt.Errorf("cannot read state db info; %v", err)
		}

		lastBlock = primeDbInfo.Block
	}

	// first block must be exactly +1 so data aligns correctly
	if lastBlock+1 != c.cfg.First {
		return fmt.Errorf("if using existing live-db with vm-sdb first block needs to be last block of live-db + 1, in your case %v", lastBlock+1)
	}

	return nil
}

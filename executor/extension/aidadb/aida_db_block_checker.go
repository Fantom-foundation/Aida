package aidadb

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"

	"github.com/Fantom-foundation/Aida/utils"
)

type aidaDbBlockChecker[T any] struct {
	extension.NilExtension[T]
	cfg         *utils.Config
	first, last uint64
}

// MakeAidaDbBlockChecker creates an executor.Extension whether given AidaDb has substate in given block range
func MakeAidaDbBlockChecker[T any](cfg *utils.Config) executor.Extension[T] {
	if cfg.AidaDb == "" {
		return extension.NilExtension[T]{}
	}

	first, last, _ := utils.FindBlockRangeInSubstate()
	return makeAidaDbBlockChecker[T](cfg, first, last)
}

func makeAidaDbBlockChecker[T any](cfg *utils.Config, first, last uint64) executor.Extension[T] {
	return &aidaDbBlockChecker[T]{
		cfg:   cfg,
		first: first,
		last:  last,
	}
}

// PreRun checks whether the block range of AidaDb and StateDb aligns.
func (c *aidaDbBlockChecker[T]) PreRun(executor.State[T], *executor.Context) error {
	if c.first == 0 && c.last == 0 {
		return fmt.Errorf("your aida-db does not have substate")
	}

	if c.first > c.cfg.First {
		return fmt.Errorf("first block of given aida-db (%v) is larger than given first block (%v), please choose first block in range", c.first, c.cfg.First)
	}
	if c.last < c.cfg.Last {
		return fmt.Errorf("last block of given aida-db (%v) is smaller than given last block (%v), please choose last block in range", c.last, c.cfg.Last)
	}

	return nil
}

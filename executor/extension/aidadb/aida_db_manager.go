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

package aidadb

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

// MakeAidaDbManager opens AidaDb if path is given and adds it to the context.
func MakeAidaDbManager[T any](cfg *utils.Config) executor.Extension[T] {
	if cfg.AidaDb == "" {
		return extension.NilExtension[T]{}
	}
	return &AidaDbManager[T]{path: cfg.AidaDb}
}

type AidaDbManager[T any] struct {
	extension.NilExtension[T]
	path string
}

func (e *AidaDbManager[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	db, err := rawdb.NewLevelDBDatabase(e.path, 1024, 100, "", true)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}
	ctx.AidaDb = db

	return nil
}

func (e *AidaDbManager[T]) PostRun(_ executor.State[T], ctx *executor.Context, _ error) error {
	if err := ctx.AidaDb.Close(); err != nil {
		return fmt.Errorf("cannot close AidaDb; %v", err)
	}

	ctx.AidaDb = nil

	return nil
}

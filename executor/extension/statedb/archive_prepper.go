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

package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
)

// MakeArchivePrepper creates an extension for retrieving archive. Archive is assigned to context.Archive.
func MakeArchivePrepper[T any]() executor.Extension[T] {
	return &archivePrepper[T]{}
}

type archivePrepper[T any] struct {
	extension.NilExtension[T]
}

// PreBlock sends needed archive to the processor.
func (r *archivePrepper[T]) PreBlock(state executor.State[T], ctx *executor.Context) error {
	var err error
	ctx.Archive, err = ctx.State.GetArchiveState(uint64(state.Block) - 1)
	if err != nil {
		return err
	}

	return nil
}

func (r *archivePrepper[T]) PreTransaction(state executor.State[T], ctx *executor.Context) error {
	return ctx.Archive.BeginTransaction(uint32(state.Transaction))
}

func (r *archivePrepper[T]) PostTransaction(_ executor.State[T], ctx *executor.Context) error {
	return ctx.Archive.EndTransaction()
}

// PostBlock releases the Archive StateDb
func (r *archivePrepper[T]) PostBlock(_ executor.State[T], ctx *executor.Context) error {
	ctx.Archive.Release()
	return nil
}

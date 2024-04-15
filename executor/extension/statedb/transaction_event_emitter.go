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

// MakeTransactionEventEmitter creates a executor.Extension to call BeginBlock() and EndBlock()
func MakeTransactionEventEmitter[T any]() executor.Extension[T] {
	return &transactionEventEmitter[T]{}
}

type transactionEventEmitter[T any] struct {
	extension.NilExtension[T]
}

func (transactionEventEmitter[T]) PreTransaction(state executor.State[T], ctx *executor.Context) error {
	return ctx.State.BeginTransaction(uint32(state.Transaction))
}

func (transactionEventEmitter[T]) PostTransaction(_ executor.State[T], ctx *executor.Context) error {
	return ctx.State.EndTransaction()
}

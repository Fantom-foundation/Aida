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
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/txcontext"
)

// MakeStateDbPrepper creates an executor extension calling PrepareSubstate on
// an optional StateDB instance before each transaction of an execution. Its main
// purpose is to support Aida's in-memory DB implementation by feeding it data
// information before each transaction in tools like `aida-vm-sdb`.
func MakeStateDbPrepper() executor.Extension[txcontext.TxContext] {
	return &statePrepper{}
}

type statePrepper struct {
	extension.NilExtension[txcontext.TxContext]
}

func (e *statePrepper) PreTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	if ctx != nil && ctx.State != nil && state.Data != nil {
		alloc := state.Data.GetInputState()
		ctx.State.PrepareSubstate(alloc, uint64(state.Block))
	}
	return nil
}

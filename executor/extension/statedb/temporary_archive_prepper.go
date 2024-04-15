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
	"github.com/Fantom-foundation/Aida/rpc"
)

// MakeTemporaryArchivePrepper creates an extension for retrieving temporary archive before every txcontext.
// Archive is assigned to context.Archive. Archive is released after transaction.
func MakeTemporaryArchivePrepper() executor.Extension[*rpc.RequestAndResults] {
	return &temporaryArchivePrepper{}
}

type temporaryArchivePrepper struct {
	extension.NilExtension[*rpc.RequestAndResults]
}

// PreTransaction creates temporary archive that is released after transaction is executed.
func (r *temporaryArchivePrepper) PreTransaction(state executor.State[*rpc.RequestAndResults], ctx *executor.Context) error {
	var err error
	ctx.Archive, err = ctx.State.GetArchiveState(uint64(state.Data.RequestedBlock))
	if err != nil {
		return err
	}

	return nil
}

// PostTransaction releases temporary Archive.
func (r *temporaryArchivePrepper) PostTransaction(_ executor.State[*rpc.RequestAndResults], ctx *executor.Context) error {
	ctx.Archive.Release()

	return nil
}

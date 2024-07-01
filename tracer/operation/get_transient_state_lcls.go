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

package operation

import (
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/context"
)

// The GetTransientStateLcls operation is a GetState operation
// whose addresses refer to previous recorded/replayed
// operations.
// (NB: Lc = last contract addreess, ls = last storage
// address)

// GetTransientStateLcls data structure
type GetTransientStateLcls struct {
}

// GetId returns the get-state-transient-lcls operation identifier.
func (op *GetTransientStateLcls) GetId() byte {
	return GetTransientStateLclsID
}

// NewGetTransientStateLcls creates a new get-state-transient-lcls operation.
func NewGetTransientStateLcls() *GetTransientStateLcls {
	return new(GetTransientStateLcls)
}

// ReadGetTransientStateLcls reads a get-state-transient-lcls operation from a file.
func ReadGetTransientStateLcls(f io.Reader) (Operation, error) {
	return NewGetTransientStateLcls(), nil
}

// Write the get-state-transient-lcls operation to file.
func (op *GetTransientStateLcls) Write(f io.Writer) error {
	return nil
}

// Execute the get-state-transient-lcls operation.
func (op *GetTransientStateLcls) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.PrevContract()
	storage := ctx.DecodeKeyCache(0)
	start := time.Now()
	db.GetState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-state-transient-lcls operation.
func (op *GetTransientStateLcls) Debug(ctx *context.Context) {
	contract := ctx.PrevContract()
	storage := ctx.ReadKeyCache(0)
	fmt.Print(contract, storage)
}

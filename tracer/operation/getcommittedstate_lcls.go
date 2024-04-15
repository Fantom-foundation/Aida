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

package operation

import (
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// GetCommittedStateLcls is a GetCommittedState operation
// whose contract/storage addresses refer to previously
// recorded/replayed operations.
// (NB: Lc = last contract address, ls = last storage
// address)

// GetCommittedStateLcls data structure
type GetCommittedStateLcls struct {
}

// GetId returns the get-committed-state-lcls operation identifier.
func (op *GetCommittedStateLcls) GetId() byte {
	return GetCommittedStateLclsID
}

// NewGetCommittedStateLcls creates a new get-committed-state-lcls operation.
func NewGetCommittedStateLcls() *GetCommittedStateLcls {
	return new(GetCommittedStateLcls)
}

// ReadGetCommittedStateLcls reads a get-committed-state-lcls operation from a file.
func ReadGetCommittedStateLcls(f io.Reader) (Operation, error) {
	return NewGetCommittedStateLcls(), nil
}

// Write the get-committed-state-lcls operation to file.
func (op *GetCommittedStateLcls) Write(f io.Writer) error {
	return nil
}

// Execute the get-committed-state-lcls operation.
func (op *GetCommittedStateLcls) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.PrevContract()
	storage := ctx.DecodeKeyCache(0)
	start := time.Now()
	db.GetCommittedState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-committed-state-lcls operation.
func (op *GetCommittedStateLcls) Debug(ctx *context.Context) {
	contract := ctx.PrevContract()
	storage := ctx.ReadKeyCache(0)
	fmt.Print(contract, storage)
}

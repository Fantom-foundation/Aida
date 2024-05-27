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

// GetCodeHashLc is a GetCodeHash operations whose
// contract address refers to previously recorded/
// replayed operations.
// (NB: Lc = last contract address)

// GetCodeHashLc data structure
type GetCodeHashLc struct {
}

// GetId returns the get-code-hash-lc operation identifier.
func (op *GetCodeHashLc) GetId() byte {
	return GetCodeHashLcID
}

// NewGetCodeHashLc creates a new get-code-hash-lc operation.
func NewGetCodeHashLc() *GetCodeHashLc {
	return &GetCodeHashLc{}
}

// ReadGetCodeHashLc reads a get-code-hash-lc operation from a file.
func ReadGetCodeHashLc(f io.Reader) (Operation, error) {
	return NewGetCodeHashLc(), nil
}

// Write the get-code-hash-lc operation to a file.
func (op *GetCodeHashLc) Write(f io.Writer) error {
	return nil
}

// Execute the get-code-hash-lc operation.
func (op *GetCodeHashLc) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.PrevContract()
	start := time.Now()
	db.GetCodeHash(contract)
	return time.Since(start)
}

// Debug prints a debug message for the get-code-hash-lc operation.
func (op *GetCodeHashLc) Debug(ctx *context.Context) {
	contract := ctx.PrevContract()
	fmt.Print(contract)
}

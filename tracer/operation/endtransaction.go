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
	"encoding/binary"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// EndTransaction data structure
type EndTransaction struct {
}

// GetId returns the end-transaction operation identifier.
func (op *EndTransaction) GetId() byte {
	return EndTransactionID
}

// NewEndTransaction creates a new end-transaction operation.
func NewEndTransaction() *EndTransaction {
	return &EndTransaction{}
}

// ReadEndTransaction reads a new end-transaction operation from file.
func ReadEndTransaction(io.Reader) (Operation, error) {
	return new(EndTransaction), nil
}

// Write the end-transaction operation to file.
func (op *EndTransaction) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the end-transaction operation.
func (op *EndTransaction) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	ctx.InitSnapshot()
	start := time.Now()
	db.EndTransaction()
	return time.Since(start)
}

// Debug prints a debug message for the end-transaction operation.
func (op *EndTransaction) Debug(*context.Context) {
}

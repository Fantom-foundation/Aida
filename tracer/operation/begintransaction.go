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
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// BeginTransaction data structure
type BeginTransaction struct {
	TransactionNumber uint32 // transaction number
}

// GetId returns the begin-transaction operation identifier.
func (op *BeginTransaction) GetId() byte {
	return BeginTransactionID
}

// NewBeginTransaction creates a new begin-transaction operation.
func NewBeginTransaction(tx uint32) *BeginTransaction {
	return &BeginTransaction{tx}
}

// ReadBeginTransaction reads a new begin-transaction operation from file.
func ReadBeginTransaction(f io.Reader) (Operation, error) {
	data := new(BeginTransaction)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the begin-transaction operation to file.
func (op *BeginTransaction) Write(f io.Writer) error {
	return binary.Write(f, binary.LittleEndian, *op)
}

// Execute the begin-transaction operation.
func (op *BeginTransaction) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	start := time.Now()
	db.BeginTransaction(op.TransactionNumber)
	return time.Since(start)
}

// Debug prints a debug message for the begin-transaction operation.
func (op *BeginTransaction) Debug(*context.Context) {
	fmt.Print(op.TransactionNumber)
}

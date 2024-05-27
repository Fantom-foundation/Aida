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
	"fmt"
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// Begin-block operation data structure
type BeginBlock struct {
	BlockNumber uint64 // block number
}

// GetId returns the begin-block operation identifier.
func (op *BeginBlock) GetId() byte {
	return BeginBlockID
}

// NewBeginBlock creates a new begin-block operation.
func NewBeginBlock(bbNum uint64) *BeginBlock {
	return &BeginBlock{BlockNumber: bbNum}
}

// ReadBeginBlock reads a begin-block operation from file.
func ReadBeginBlock(f io.Reader) (Operation, error) {
	data := new(BeginBlock)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the begin-block operation to file.
func (op *BeginBlock) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the begin-block operation.
func (op *BeginBlock) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	start := time.Now()
	db.BeginBlock(op.BlockNumber)
	return time.Since(start)
}

// Debug prints a debug message for the begin-block operation.
func (op *BeginBlock) Debug(ctx *context.Context) {
	fmt.Print(op.BlockNumber)
}

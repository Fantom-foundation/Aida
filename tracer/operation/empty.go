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
	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// Empty data structure
type Empty struct {
	Contract common.Address
}

// GetId returns the Empty operation identifier.
func (op *Empty) GetId() byte {
	return EmptyID
}

// NewEmpty creates a new Empty operation.
func NewEmpty(contract common.Address) *Empty {
	return &Empty{Contract: contract}
}

// ReadEmpty reads an Empty operation from a file.
func ReadEmpty(f io.Reader) (Operation, error) {
	data := new(Empty)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the Empty operation to a file.
func (op *Empty) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the Empty operation.
func (op *Empty) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.Empty(contract)
	return time.Since(start)
}

// Debug prints a debug message for the Empty operation.
func (op *Empty) Debug(ctx *context.Context) {
	fmt.Print(op.Contract)
}

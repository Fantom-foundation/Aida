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
	"github.com/ethereum/go-ethereum/common"
)

// HasSelfDestructed data structure
type HasSelfDestructed struct {
	Contract common.Address
}

// GetId returns the HasSelfDestructed operation identifier.
func (op *HasSelfDestructed) GetId() byte {
	return HasSelfDestructedID
}

// NewHasSelfDestructed creates a new HasSelfDestructed operation.
func NewHasSelfDestructed(contract common.Address) *HasSelfDestructed {
	return &HasSelfDestructed{Contract: contract}
}

// ReadHasSelfDestructed reads a HasSelfDestructed operation from a file.
func ReadHasSelfDestructed(f io.Reader) (Operation, error) {
	data := new(HasSelfDestructed)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the HasSelfDestructed operation to a file.
func (op *HasSelfDestructed) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the HasSelfDestructed operation.
func (op *HasSelfDestructed) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.HasSelfDestructed(contract)
	return time.Since(start)
}

// Debug prints a debug message for the HasSelfDestructed operation.
func (op *HasSelfDestructed) Debug(ctx *context.Context) {
	fmt.Print(op.Contract)
}

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

// HasSuicided data structure
type HasSuicided struct {
	Contract common.Address
}

// GetId returns the HasSuicided operation identifier.
func (op *HasSuicided) GetId() byte {
	return HasSuicidedID
}

// NewHasSuicided creates a new HasSuicided operation.
func NewHasSuicided(contract common.Address) *HasSuicided {
	return &HasSuicided{Contract: contract}
}

// ReadHasSuicided reads a HasSuicided operation from a file.
func ReadHasSuicided(f io.Reader) (Operation, error) {
	data := new(HasSuicided)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the HasSuicided operation to a file.
func (op *HasSuicided) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the HasSuicided operation.
func (op *HasSuicided) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.HasSelfDestructed(contract)
	return time.Since(start)
}

// Debug prints a debug message for the HasSuicided operation.
func (op *HasSuicided) Debug(ctx *context.Context) {
	fmt.Print(op.Contract)
}

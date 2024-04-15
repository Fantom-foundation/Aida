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
	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// Suicide data structure
type Suicide struct {
	Contract common.Address
}

// GetId returns the suicide operation identifier.
func (op *Suicide) GetId() byte {
	return SuicideID
}

// NewSuicide creates a new suicide operation.
func NewSuicide(contract common.Address) *Suicide {
	return &Suicide{Contract: contract}
}

// ReadSuicide reads a suicide operation from a file.
func ReadSuicide(f io.Reader) (Operation, error) {
	data := new(Suicide)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the suicide operation to a file.
func (op *Suicide) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the suicide operation.
func (op *Suicide) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.Suicide(contract)
	return time.Since(start)
}

// Debug prints a debug message for the suicide operation.
func (op *Suicide) Debug(ctx *context.Context) {
	fmt.Print(ctx.DecodeContract(op.Contract))
}

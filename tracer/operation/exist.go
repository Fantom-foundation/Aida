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

// Exist data structure
type Exist struct {
	Contract common.Address
}

// GetId returns the exist operation identifier.
func (op *Exist) GetId() byte {
	return ExistID
}

// NewExist creates a new exist operation.
func NewExist(contract common.Address) *Exist {
	return &Exist{Contract: contract}
}

// ReadExist reads an exist operation from a file.
func ReadExist(f io.Reader) (Operation, error) {
	data := new(Exist)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the exist operation to a file.
func (op *Exist) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the exist operation.
func (op *Exist) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.Exist(contract)
	return time.Since(start)
}

// Debug prints a debug message for the exist operation.
func (op *Exist) Debug(ctx *context.Context) {
	fmt.Print(ctx.DecodeContract(op.Contract))
}

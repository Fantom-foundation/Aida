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

// GetCode data structure
type GetCode struct {
	Contract common.Address
}

// GetId returns the get-code operation identifier.
func (op *GetCode) GetId() byte {
	return GetCodeID
}

// NewGetCode creates a new get-code operation.
func NewGetCode(contract common.Address) *GetCode {
	return &GetCode{Contract: contract}
}

// ReadGetCode reads a get-code operation from a file.
func ReadGetCode(f io.Reader) (Operation, error) {
	data := new(GetCode)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-code operation to a file.
func (op *GetCode) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-code operation.
func (op *GetCode) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.GetCode(contract)
	return time.Since(start)
}

// Debug prints a debug message for the get-code operation.
func (op *GetCode) Debug(ctx *context.Context) {
	fmt.Print(op.Contract)
}

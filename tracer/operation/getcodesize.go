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

// GetCodeSize data structure
type GetCodeSize struct {
	Contract common.Address
}

// GetCodeSize returns the get-code-size operation identifier.
func (op *GetCodeSize) GetId() byte {
	return GetCodeSizeID
}

// NewGetCodeSize creates a new get-code-size operation.
func NewGetCodeSize(contract common.Address) *GetCodeSize {
	return &GetCodeSize{Contract: contract}
}

// ReadGetCodeSize reads a get-code-size operation from a file.
func ReadGetCodeSize(f io.Reader) (Operation, error) {
	data := new(GetCodeSize)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-code-size operation to a file.
func (op *GetCodeSize) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-code-size operation.
func (op *GetCodeSize) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.GetCodeSize(contract)
	return time.Since(start)
}

// Debug prints a debug message for get-code-size.
func (op *GetCodeSize) Debug(ctx *context.Context) {
	fmt.Print(ctx.DecodeContract(op.Contract))
}

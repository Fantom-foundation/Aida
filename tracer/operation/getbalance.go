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

// GetBalance data structure
type GetBalance struct {
	Contract common.Address
}

// GetId returns the get-balance operation identifier.
func (op *GetBalance) GetId() byte {
	return GetBalanceID
}

// NewGetBalance creates a new get-balance operation.
func NewGetBalance(contract common.Address) *GetBalance {
	return &GetBalance{Contract: contract}
}

// ReadGetBalance reads a get-balance operation from a file.
func ReadGetBalance(f io.Reader) (Operation, error) {
	data := new(GetBalance)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-balance operation.
func (op *GetBalance) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-balance operation.
func (op *GetBalance) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.GetBalance(contract)
	return time.Since(start)
}

// Debug prints a debug message for the get-balance operation.
func (op *GetBalance) Debug(ctx *context.Context) {
	fmt.Print(op.Contract)
}

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

// GetState data structure
type GetState struct {
	Contract common.Address
	Key      common.Hash
}

// GetId returns the get-state operation identifier.
func (op *GetState) GetId() byte {
	return GetStateID
}

// NewGetState creates a new get-state operation.
func NewGetState(contract common.Address, key common.Hash) *GetState {
	return &GetState{Contract: contract, Key: key}
}

// ReadGetState reads a get-state operation from a file.
func ReadGetState(f io.Reader) (Operation, error) {
	data := new(GetState)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-state operation to file.
func (op *GetState) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-state operation.
func (op *GetState) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	storage := ctx.DecodeKey(op.Key)
	start := time.Now()
	db.GetState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-state operation.
func (op *GetState) Debug(ctx *context.Context) {
	fmt.Print(op.Contract, op.Key)
}

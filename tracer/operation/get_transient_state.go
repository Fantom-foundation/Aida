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

// GetTransientState data structure
type GetTransientState struct {
	Contract common.Address
	Key      common.Hash
}

// GetId returns the get-transient-state operation identifier.
func (op *GetTransientState) GetId() byte {
	return GetTransientStateID
}

// NewGetTransientState creates a new get-transient-state operation.
func NewGetTransientState(contract common.Address, key common.Hash) *GetTransientState {
	return &GetTransientState{Contract: contract, Key: key}
}

// ReadGetTransientState reads a get-transient-state operation from a file.
func ReadGetTransientState(f io.Reader) (Operation, error) {
	data := new(GetTransientState)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-transient-state operation to file.
func (op *GetTransientState) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-transient-state operation.
func (op *GetTransientState) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	storage := ctx.DecodeKey(op.Key)
	start := time.Now()
	db.GetTransientState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-transient-state operation.
func (op *GetTransientState) Debug(ctx *context.Context) {
	fmt.Print(op.Contract, op.Key)
}

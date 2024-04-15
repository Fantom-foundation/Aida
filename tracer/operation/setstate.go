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

// SetState data structure
type SetState struct {
	Contract common.Address // encoded contract address
	Key      common.Hash    // encoded storage address
	Value    common.Hash    // encoded storage value
}

// GetId returns the set-state identifier.
func (op *SetState) GetId() byte {
	return SetStateID
}

// NewSetState creates a new set-state operation.
func NewSetState(contract common.Address, key common.Hash, value common.Hash) *SetState {
	return &SetState{Contract: contract, Key: key, Value: value}
}

// ReadSetState reads a set-state operation from file.
func ReadSetState(f io.Reader) (Operation, error) {
	data := new(SetState)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the set-state operation to file.
func (op *SetState) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the set-state operation.
func (op *SetState) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	storage := ctx.DecodeKey(op.Key)
	value := op.Value
	start := time.Now()
	db.SetState(contract, storage, value)
	return time.Since(start)
}

// Debug prints a debug message for the set-state operation.
func (op *SetState) Debug(ctx *context.Context) {
	fmt.Print(op.Contract, op.Key, op.Value)
}

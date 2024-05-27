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

// GetCommittedState data structure
type GetCommittedState struct {
	Contract common.Address
	Key      common.Hash
}

// GetId returns the get-commited-state-operation identifier.
func (op *GetCommittedState) GetId() byte {
	return GetCommittedStateID
}

// NewGetCommittedState creates a new get-commited-state operation.
func NewGetCommittedState(contract common.Address, key common.Hash) *GetCommittedState {
	return &GetCommittedState{Contract: contract, Key: key}
}

// ReadGetCommittedState reads a get-commited-state operation from file.
func ReadGetCommittedState(f io.Reader) (Operation, error) {
	data := new(GetCommittedState)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-commited-state operation to file.
func (op *GetCommittedState) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-committed-state operation.
func (op *GetCommittedState) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	storage := ctx.DecodeKey(op.Key)
	start := time.Now()
	db.GetCommittedState(contract, storage)
	return time.Since(start)
}

// Debug prints debug message for the get-committed-state operation.
func (op *GetCommittedState) Debug(ctx *context.Context) {
	fmt.Print(op.Contract, op.Key)
}

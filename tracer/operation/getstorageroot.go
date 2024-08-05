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

// GetStorageRoot data structure
type GetStorageRoot struct {
	Contract common.Address
}

// GetId returns the get-storage-root operation identifier.
func (op *GetStorageRoot) GetId() byte {
	return GetStorageRootID
}

// NewGetStorageRoot creates a new get-storage-root operation.
func NewGetStorageRoot(contract common.Address) *GetStorageRoot {
	return &GetStorageRoot{Contract: contract}
}

// ReadGetHash reads a get-storage-root operation from a file.
func ReadGetStorageRoot(f io.Reader) (Operation, error) {
	data := new(GetStorageRoot)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-storage-root operation to a file.
func (op *GetStorageRoot) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-storage-root operation.
func (op *GetStorageRoot) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.GetStorageRoot(contract)
	return time.Since(start)
}

// Debug prints a debug message for the get-storage-root operation.
func (op *GetStorageRoot) Debug(ctx *context.Context) {
	fmt.Print(op.Contract)
}

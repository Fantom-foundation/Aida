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

// The GetStateLc operation is a GetState operation
// whose contract address refers to previously
// recorded/replayed operations.
// (NB: Lc = last contract addresss used)

// GetStateLc data structure
type GetStateLc struct {
	Key common.Hash
}

// GetId returns the get-state-lc operation identifier.
func (op *GetStateLc) GetId() byte {
	return GetStateLcID
}

// GetStateLc creates a new get-state-lc operation.
func NewGetStateLc(key common.Hash) *GetStateLc {
	return &GetStateLc{Key: key}
}

// ReadGetStateLc reads a get-state-lc operation from a file.
func ReadGetStateLc(f io.Reader) (Operation, error) {
	data := new(GetStateLc)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-state-lc operation to file.
func (op *GetStateLc) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-state-lc operation.
func (op *GetStateLc) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.PrevContract()
	storage := ctx.DecodeKey(op.Key)
	start := time.Now()
	db.GetState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-state-lc operation.
func (op *GetStateLc) Debug(ctx *context.Context) {
	contract := ctx.PrevContract()
	storage := ctx.DecodeKey(op.Key)
	fmt.Print(contract, storage)
}

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

// The GetTransientStateLc operation is a GetState operation
// whose contract address refers to previously
// recorded/replayed operations.
// (NB: Lc = last contract addresss used)

// GetTransientStateLc data structure
type GetTransientStateLc struct {
	Key common.Hash
}

// GetId returns the get-transient-state-lc operation identifier.
func (op *GetTransientStateLc) GetId() byte {
	return GetTransientStateLcID
}

// GetTransientStateLc creates a new get-transient-state-lc operation.
func NewGetTransientStateLc(key common.Hash) *GetTransientStateLc {
	return &GetTransientStateLc{Key: key}
}

// ReadGetTransientStateLc reads a get-transient-state-lc operation from a file.
func ReadGetTransientStateLc(f io.Reader) (Operation, error) {
	data := new(GetTransientStateLc)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-transient-state-lc operation to file.
func (op *GetTransientStateLc) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-transient-state-lc operation.
func (op *GetTransientStateLc) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.PrevContract()
	storage := ctx.DecodeKey(op.Key)
	start := time.Now()
	db.GetState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-transient-state-lc operation.
func (op *GetTransientStateLc) Debug(ctx *context.Context) {
	contract := ctx.PrevContract()
	storage := ctx.DecodeKey(op.Key)
	fmt.Print(contract, storage)
}

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

// GetNonce data structure
type GetNonce struct {
	Contract common.Address
}

// GetId returns the get-nonce operation identifier.
func (op *GetNonce) GetId() byte {
	return GetNonceID
}

// NewGetNonce creates a new get-nonce operation.
func NewGetNonce(contract common.Address) *GetNonce {
	return &GetNonce{Contract: contract}
}

// ReadGetNonce reads a get-nonce operation from a file.
func ReadGetNonce(f io.Reader) (Operation, error) {
	data := new(GetNonce)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-nonce operation to a file.
func (op *GetNonce) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-nonce operation.
func (op *GetNonce) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.GetNonce(contract)
	return time.Since(start)
}

// Debug prints a debug message for the get-nonce operation.
func (op *GetNonce) Debug(ctx *context.Context) {
	fmt.Print(op.Contract)
}

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
	"log"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/holiman/uint256"
)

// AddBalance data structure
type AddBalance struct {
	Contract common.Address
	Amount   [16]byte // truncated amount to 16 bytes
}

// GetId returns the add-balance operation identifier.
func (op *AddBalance) GetId() byte {
	return AddBalanceID
}

// NewAddBalance creates a new add-balance operation.
func NewAddBalance(contract common.Address, amount *uint256.Int) *AddBalance {
	// check if amount requires more than 256 bits (16 bytes)
	if amount.BitLen() > 256 {
		log.Fatalf("Amount exceeds 256 bit")
	}
	ret := &AddBalance{Contract: contract}
	// copy amount to a 16-byte array with leading zeros
	amount.SetBytes(ret.Amount[:])
	return ret
}

// ReadAddBalance reads a add-balance operation from a file.
func ReadAddBalance(f io.Reader) (Operation, error) {
	data := new(AddBalance)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write writes the add-balance operation to a file.
func (op *AddBalance) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute executes the add-balance operation.
func (op *AddBalance) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	// construct bit.Int from a byte array
	amount := new(uint256.Int).SetBytes(op.Amount[:])
	start := time.Now()
	// ignore reason
	db.AddBalance(contract, amount, 0)
	return time.Since(start)
}

// Debug prints a debug message for the add-balance operation.
func (op *AddBalance) Debug(ctx *context.Context) {
	fmt.Print(op.Contract, new(uint256.Int).SetBytes(op.Amount[:]))
}

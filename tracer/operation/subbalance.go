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

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
)

// SubBalance data structure
type SubBalance struct {
	Contract common.Address
	Amount   [32]byte // truncated amount to 32 bytes
	Reason   tracing.BalanceChangeReason
}

// GetId returns the sub-balance operation identifier.
func (op *SubBalance) GetId() byte {
	return SubBalanceID
}

// NewSubBalance creates a new sub-balance operation.
func NewSubBalance(contract common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) *SubBalance {
	// check if amount requires more than 32 bytes
	if amount.ByteLen() > 32 {
		log.Fatalf("Amount exceeds 32 bytes")
	}
	ret := &SubBalance{Contract: contract, Amount: amount.Bytes32(), Reason: reason}
	return ret
}

// ReadSubBalance reads a sub-balance operation from a file.
func ReadSubBalance(f io.Reader) (Operation, error) {
	data := new(SubBalance)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the sub-balance operation to a file.
func (op *SubBalance) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the sub-balance operation.
func (op *SubBalance) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	// construct bit.Int from a byte array
	amount := new(uint256.Int).SetBytes(op.Amount[:])
	start := time.Now()
	// Ignore reason
	db.SubBalance(contract, amount, op.Reason)
	return time.Since(start)
}

// Debug prints a debug message for the sub-balance operation.
func (op *SubBalance) Debug(ctx *context.Context) {
	fmt.Print(op.Contract, new(uint256.Int).SetBytes(op.Amount[:]), op.Reason)
}

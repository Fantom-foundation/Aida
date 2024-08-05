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

// SelfDestruct6780 data structure
type SelfDestruct6780 struct {
	Contract common.Address
}

// GetId returns the self-destruct operation identifier.
func (op *SelfDestruct6780) GetId() byte {
	return SelfDestruct6780ID
}

// NewSelfDestruct6780 creates a new self-destruct operation.
func NewSelfDestruct6780(contract common.Address) *SelfDestruct6780 {
	return &SelfDestruct6780{Contract: contract}
}

// ReadSelfDestruct6780 reads a self-destruct operation from a file.
func ReadSelfDestruct6780(f io.Reader) (Operation, error) {
	data := new(SelfDestruct6780)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the self-destruct operation to a file.
func (op *SelfDestruct6780) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the self-destruct operation.
func (op *SelfDestruct6780) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.Selfdestruct6780(contract)
	return time.Since(start)
}

// Debug prints a debug message for the self-destruct operation.
func (op *SelfDestruct6780) Debug(ctx *context.Context) {
	fmt.Print(ctx.DecodeContract(op.Contract))
}

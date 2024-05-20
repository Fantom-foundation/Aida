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

// SetCode data structure
type SetCode struct {
	Contract common.Address
	Bytecode []byte // encoded bytecode
}

// GetId returns the set-code operation identifier.
func (op *SetCode) GetId() byte {
	return SetCodeID
}

// NewSetCode creates a new set-code operation.
func NewSetCode(contract common.Address, bytecode []byte) *SetCode {
	return &SetCode{Contract: contract, Bytecode: bytecode}
}

// ReadSetCode reads a set-code operation from a file.
func ReadSetCode(f io.Reader) (Operation, error) {
	data := new(SetCode)
	if err := binary.Read(f, binary.LittleEndian, &data.Contract); err != nil {
		return nil, fmt.Errorf("Cannot read contract address. Error: %v", err)
	}
	var length uint32
	if err := binary.Read(f, binary.LittleEndian, &length); err != nil {
		return nil, fmt.Errorf("Cannot read byte-code length. Error: %v", err)
	}
	data.Bytecode = make([]byte, length)
	if err := binary.Read(f, binary.LittleEndian, data.Bytecode); err != nil {
		return nil, fmt.Errorf("Cannot read byte-code. Error: %v", err)
	}
	return data, nil
}

// Write the set-code operation to a file.
func (op *SetCode) Write(f io.Writer) error {
	if err := binary.Write(f, binary.LittleEndian, op.Contract); err != nil {
		return fmt.Errorf("Cannot write contract address. Error: %v", err)
	}
	var length = uint32(len(op.Bytecode))
	if err := binary.Write(f, binary.LittleEndian, &length); err != nil {
		return fmt.Errorf("Cannot read byte-code length. Error: %v", err)
	}
	if err := binary.Write(f, binary.LittleEndian, op.Bytecode); err != nil {
		return fmt.Errorf("Cannot write byte-code. Error: %v", err)
	}
	return nil
}

// Execute the set-code operation.
func (op *SetCode) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.SetCode(contract, op.Bytecode)
	return time.Since(start)
}

// Debug prints a debug message for the set-code operation.
func (op *SetCode) Debug(ctx *context.Context) {
	fmt.Print(op.Contract, op.Bytecode)
}

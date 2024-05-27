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

// CreateAccount data structure
type CreateAccount struct {
	Contract common.Address
}

// GetId returns the create-account operation identifier.
func (op *CreateAccount) GetId() byte {
	return CreateAccountID
}

// NewCreateAcccount creates a new create-account operation.
func NewCreateAccount(contract common.Address) *CreateAccount {
	return &CreateAccount{Contract: contract}
}

// ReadCreateAccount reads a create-account operation from a file.
func ReadCreateAccount(f io.Reader) (Operation, error) {
	data := new(CreateAccount)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the create-account operation to file.
func (op *CreateAccount) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the create-account operation.
func (op *CreateAccount) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.DecodeContract(op.Contract)
	start := time.Now()
	db.CreateAccount(contract)
	return time.Since(start)
}

// Debug prints a debug message for the create-account operation.
func (op *CreateAccount) Debug(ctx *context.Context) {
	fmt.Print(op.Contract)
}

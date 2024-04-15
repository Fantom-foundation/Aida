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

// The SetStateLcls operation is a SetState operation whose
// addresses refer to previously recorded/replayed addresses.
// (NB: Lc = last contract address, Ls = last storage address)

// SetStateLcls data structure
type SetStateLcls struct {
	Value common.Hash // encoded storage value
}

// GetId returns the set-state-lcls identifier.
func (op *SetStateLcls) GetId() byte {
	return SetStateLclsID
}

// SetStateLcls creates a new set-state-lcls operation.
func NewSetStateLcls(value common.Hash) *SetStateLcls {
	return &SetStateLcls{Value: value}
}

// ReadSetStateLcls reads a set-state-lcls operation from file.
func ReadSetStateLcls(f io.Reader) (Operation, error) {
	data := new(SetStateLcls)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the set-state-lcls operation to file.
func (op *SetStateLcls) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the set-state-lcls operation.
func (op *SetStateLcls) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.PrevContract()
	storage := ctx.DecodeKeyCache(0)
	start := time.Now()
	db.SetState(contract, storage, op.Value)
	return time.Since(start)
}

// Debug prints a debug message for the set-state-lcls operation.
func (op *SetStateLcls) Debug(ctx *context.Context) {
	contract := ctx.PrevContract()
	storage := ctx.ReadKeyCache(0)
	fmt.Print(contract, storage, op.Value)
}

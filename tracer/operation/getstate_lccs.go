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
)

// The GetStateLccs operation is a GetState operation whose
// addresses refer to previously recorded/replayed operations.
// (NB: Lc = last contract address, cs = cached storage
// address referring to a position in an indexed cache
// for storage addresses.)

// GetStateLccs  data structure
type GetStateLccs struct {
	StoragePosition uint8 // position in storage index-cache
}

// GetId returns the get-state-lccs operation identifier.
func (op *GetStateLccs) GetId() byte {
	return GetStateLccsID
}

// NewGetStateLccs creates a new get-state-lccs operation.
func NewGetStateLccs(sPos int) *GetStateLccs {
	if sPos < 0 || sPos > 255 {
		log.Fatalf("Position out of range")
	}
	return &GetStateLccs{StoragePosition: uint8(sPos)}
}

// ReadGetStateLccs reads a get-state-lccs operation from a file.
func ReadGetStateLccs(f io.Reader) (Operation, error) {
	data := new(GetStateLccs)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-state-lccs operation to file.
func (op *GetStateLccs) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-state-lccs operation.
func (op *GetStateLccs) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.PrevContract()
	storage := ctx.DecodeKeyCache(int(op.StoragePosition))
	start := time.Now()
	db.GetState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-state-lccs operation.
func (op *GetStateLccs) Debug(ctx *context.Context) {
	contract := ctx.PrevContract()
	storage := ctx.ReadKeyCache(int(op.StoragePosition))
	fmt.Print(contract, storage)
}

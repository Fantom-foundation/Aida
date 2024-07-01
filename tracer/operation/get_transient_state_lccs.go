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

// The GetTransientStateLccs operation is a GetState operation whose
// addresses refer to previously recorded/replayed operations.
// (NB: Lc = last contract address, cs = cached storage
// address referring to a position in an indexed cache
// for storage addresses.)

// GetTransientStateLccs  data structure
type GetTransientStateLccs struct {
	StoragePosition uint8 // position in storage index-cache
}

// GetId returns the get-state-transient-lccs operation identifier.
func (op *GetTransientStateLccs) GetId() byte {
	return GetTransientStateLccsID
}

// NewGetTransientStateLccs creates a new get-state-transient-lccs operation.
func NewGetTransientStateLccs(sPos int) *GetTransientStateLccs {
	if sPos < 0 || sPos > 255 {
		log.Fatalf("Position out of range")
	}
	return &GetTransientStateLccs{StoragePosition: uint8(sPos)}
}

// ReadGetTransientStateLccs reads a get-state-transient-lccs operation from a file.
func ReadGetTransientStateLccs(f io.Reader) (Operation, error) {
	data := new(GetTransientStateLccs)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the get-state-transient-lccs operation to file.
func (op *GetTransientStateLccs) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the get-state-transient-lccs operation.
func (op *GetTransientStateLccs) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	contract := ctx.PrevContract()
	storage := ctx.DecodeKeyCache(int(op.StoragePosition))
	start := time.Now()
	db.GetState(contract, storage)
	return time.Since(start)
}

// Debug prints a debug message for the get-state-transient-lccs operation.
func (op *GetTransientStateLccs) Debug(ctx *context.Context) {
	contract := ctx.PrevContract()
	storage := ctx.ReadKeyCache(int(op.StoragePosition))
	fmt.Print(contract, storage)
}

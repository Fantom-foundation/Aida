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

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// RevertToSnapshot data structure
type RevertToSnapshot struct {
	SnapshotID int32 // snapshot id limited to 32 bits.
}

// RevertToSnapshot returns the revert-to-snapshot operation identifier.
func (op *RevertToSnapshot) GetId() byte {
	return RevertToSnapshotID
}

// NewRevertToSnapshot creates a new revert-to-snapshot operation.
func NewRevertToSnapshot(SnapshotID int) *RevertToSnapshot {
	return &RevertToSnapshot{SnapshotID: int32(SnapshotID)}
}

// ReadRevertToSnapshot reads revert-to-snapshot operation from file.
func ReadRevertToSnapshot(f io.Reader) (Operation, error) {
	data := new(RevertToSnapshot)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the revert-to-snapshot operation to file.
func (op *RevertToSnapshot) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the revert-to-snapshot operation.
func (op *RevertToSnapshot) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	id := ctx.GetSnapshot(op.SnapshotID)
	start := time.Now()
	db.RevertToSnapshot(int(id))
	return time.Since(start)
}

// Debug prints a debug message for the revert-to-snapshot operation.
func (op *RevertToSnapshot) Debug(ctx *context.Context) {
	fmt.Print(op.SnapshotID)
}

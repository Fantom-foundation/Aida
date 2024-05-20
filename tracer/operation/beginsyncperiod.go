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

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// BeginSyncPeriod data structure
type BeginSyncPeriod struct {
	SyncPeriodNumber uint64
}

// GetId returns the begin-sync-period operation identifier.
func (op *BeginSyncPeriod) GetId() byte {
	return BeginSyncPeriodID
}

// NewBeginSyncPeriod creates a new begin-sync-period operation.
func NewBeginSyncPeriod(number uint64) *BeginSyncPeriod {
	return &BeginSyncPeriod{number}
}

// ReadBeginSyncPeriod reads a begin-sync-period operation from file.
func ReadBeginSyncPeriod(f io.Reader) (Operation, error) {
	data := new(BeginSyncPeriod)
	err := binary.Read(f, binary.LittleEndian, data)
	return data, err
}

// Write the begin-sync-period operation to file.
func (op *BeginSyncPeriod) Write(f io.Writer) error {
	return binary.Write(f, binary.LittleEndian, *op)
}

// Execute the begin-sync-period operation.
func (op *BeginSyncPeriod) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	start := time.Now()
	db.BeginSyncPeriod(op.SyncPeriodNumber)
	return time.Since(start)
}

// Debug prints a debug message for the begin-sync-period operation.
func (op *BeginSyncPeriod) Debug(ctx *context.Context) {
	fmt.Print(op.SyncPeriodNumber)
}

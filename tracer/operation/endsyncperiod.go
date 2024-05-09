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
	"io"
	"time"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

// End-sync-period operation data structure
type EndSyncPeriod struct {
}

// GetId returns the end-sync-period operation identifier.
func (op *EndSyncPeriod) GetId() byte {
	return EndSyncPeriodID
}

// NewEndSyncPeriod creates a new end-sync-period operation.
func NewEndSyncPeriod() *EndSyncPeriod {
	return &EndSyncPeriod{}
}

// ReadEndSyncPeriod reads an end-sync-period operation from file.
func ReadEndSyncPeriod(f io.Reader) (Operation, error) {
	return new(EndSyncPeriod), nil
}

// Write the end-sync-period operation to file.
func (op *EndSyncPeriod) Write(f io.Writer) error {
	err := binary.Write(f, binary.LittleEndian, *op)
	return err
}

// Execute the end-sync-period operation.
func (op *EndSyncPeriod) Execute(db state.StateDB, ctx *context.Replay) time.Duration {
	start := time.Now()
	db.EndSyncPeriod()
	return time.Since(start)
}

// Debug prints a debug message for the end-sync-period operation.
func (op *EndSyncPeriod) Debug(ctx *context.Context) {
}

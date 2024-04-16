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
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

func initSnapshot(t *testing.T) (*context.Replay, *Snapshot, int32) {
	// create context context
	ctx := context.NewReplay()

	var snapID int32 = 1
	// create new operation
	op := NewSnapshot(snapID)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SnapshotID {
		t.Fatalf("wrong ID returned")
	}
	return ctx, op, snapID
}

// TestSnapshotReadWrite writes a new Snapshot object into a buffer, reads from it,
// and checks equality.
func TestSnapshotReadWrite(t *testing.T) {
	_, op1, _ := initSnapshot(t)
	testOperationReadWrite(t, op1, ReadSnapshot)
}

// TestSnapshotDebug creates a new Snapshot object and checks its Debug message.
func TestSnapshotDebug(t *testing.T) {
	ctx, op, snapID := initSnapshot(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(snapID))
}

// TestSnapshotExecute
func TestSnapshotExecute(t *testing.T) {
	ctx, op, _ := initSnapshot(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{SnapshotID, nil}}
	mock.compareRecordings(expected, t)
}

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
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

func initRevertToSnapshot(t *testing.T) (*context.Replay, *Snapshot, *RevertToSnapshot, int32, int32) {
	// create context context
	ctx := context.NewReplay()

	var recordedID int32 = 1
	var replayedID int32 = 2

	// create new operation
	op1 := NewSnapshot(replayedID)
	// check id
	if op1.GetId() != SnapshotID {
		t.Fatalf("wrong ID returned")
	}
	if op1 == nil {
		t.Fatalf("failed to create operation")
	}
	op2 := NewRevertToSnapshot(int(recordedID))
	if op2 == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op2.GetId() != RevertToSnapshotID {
		t.Fatalf("wrong ID returned")
	}

	ctx.AddSnapshot(recordedID, replayedID)

	return ctx, op1, op2, recordedID, replayedID
}

// TestRevertToSnapshotReadWrite writes a new RevertToSnapshot object into a buffer, reads from it,
// and checks equality.
func TestRevertToSnapshotReadWrite(t *testing.T) {
	_, _, op1, _, _ := initRevertToSnapshot(t)
	testOperationReadWrite(t, op1, ReadRevertToSnapshot)
}

// TestRevertToSnapshotDebug creates a new RevertToSnapshot object and checks its Debug message.
func TestRevertToSnapshotDebug(t *testing.T) {
	ctx, _, op2, value, _ := initRevertToSnapshot(t)
	testOperationDebug(t, ctx, op2, fmt.Sprint(value))
}

// TestRevertToSnapshotExecute
func TestRevertToSnapshotExecute(t *testing.T) {
	ctx, op1, op2, _, replayedID := initRevertToSnapshot(t)

	// check execution
	mock := NewMockStateDB()
	op1.Execute(mock, ctx)
	op2.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{SnapshotID, nil}, {RevertToSnapshotID, []any{int(replayedID)}}}
	mock.compareRecordings(expected, t)
}

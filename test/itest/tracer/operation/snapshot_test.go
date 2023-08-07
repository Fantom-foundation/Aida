package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/context"
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

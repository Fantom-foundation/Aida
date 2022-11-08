package operation

import (
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"testing"
)

func initRevertToSnapshot(t *testing.T) (*dict.DictionaryContext, *Snapshot, *RevertToSnapshot, int32, int32) {
	// create dictionary context
	dict := dict.NewDictionaryContext()

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

	dict.AddSnapshot(recordedID, replayedID)

	return dict, op1, op2, recordedID, replayedID
}

// TestRevertToSnapshotReadWrite writes a new RevertToSnapshot object into a buffer, reads from it,
// and checks equality.
func TestRevertToSnapshotReadWrite(t *testing.T) {
	_, _, op1, _, _ := initRevertToSnapshot(t)
	testOperationReadWrite(t, op1, ReadRevertToSnapshot)
}

// TestRevertToSnapshotDebug creates a new RevertToSnapshot object and checks its Debug message.
func TestRevertToSnapshotDebug(t *testing.T) {
	dict, _, op2, value, _ := initRevertToSnapshot(t)
	testOperationDebug(t, dict, op2, fmt.Sprint(value))
}

// TestRevertToSnapshotExecute
func TestRevertToSnapshotExecute(t *testing.T) {
	dict, op1, op2, _, replayedID := initRevertToSnapshot(t)

	// check execution
	mock := NewMockStateDB()
	op1.Execute(mock, dict)
	op2.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{SnapshotID, nil}, {RevertToSnapshotID, []any{int(replayedID)}}}
	mock.compareRecordings(expected, t)
}

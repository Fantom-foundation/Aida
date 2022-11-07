package operation

import (
	"bytes"
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"io"
	"os"
	"testing"
)

func initSnapshot(t *testing.T) (*dict.DictionaryContext, *Snapshot, int32) {
	// create dictionary context
	dict := dict.NewDictionaryContext()

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
	return dict, op, snapID
}

// TestSnapshotReadWrite writes a new Snapshot object into a buffer, reads from it,
// and checks equality.
func TestSnapshotReadWrite(t *testing.T) {
	_, op1, _ := initSnapshot(t)
	testOperationReadWrite(t, op1, ReadSnapshot)
}

// TestSnapshotDebug creates a new Snapshot object and checks its Debug message.
func TestSnapshotDebug(t *testing.T) {
	dict, op, snapID := initSnapshot(t)

	// divert stdout to a buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// print debug message
	op.Debug(dict)

	// restore stdout
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// check debug message
	label, f := operationLabels[SnapshotID]
	if !f {
		t.Fatalf("label for %d not found", SnapshotID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %d\n", label, snapID) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestSnapshotExecute
func TestSnapshotExecute(t *testing.T) {
	dict, op, _ := initSnapshot(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{SnapshotID, nil}}
	mock.compareRecordings(expected, t)
}

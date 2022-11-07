package operation

import (
	"bytes"
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"io"
	"os"
	"reflect"
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

	op1Buffer := bytes.NewBufferString("")
	err := op1.Write(op1Buffer)
	if err != nil {
		t.Fatalf("error operation write %v", err)
	}

	// read object from buffer
	op2Buffer := bytes.NewBufferString(op1Buffer.String())
	op2, err := ReadRevertToSnapshot(op2Buffer)
	if err != nil {
		t.Fatalf("failed to read operation. Error: %v", err)
	}
	if op2 == nil {
		t.Fatalf("failed to create newly read operation from buffer")
	}
	// check equivalence
	if !reflect.DeepEqual(op1, op2) {
		t.Fatalf("operations are not the same")
	}
}

// TestRevertToSnapshotDebug creates a new RevertToSnapshot object and checks its Debug message.
func TestRevertToSnapshotDebug(t *testing.T) {
	dict, _, op2, value, _ := initRevertToSnapshot(t)

	// divert stdout to a buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// print debug message
	op2.Debug(dict)

	// restore stdout
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// check debug message
	label, f := operationLabels[RevertToSnapshotID]
	if !f {
		t.Fatalf("label for %d not found", RevertToSnapshotID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %d\n", label, value) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
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

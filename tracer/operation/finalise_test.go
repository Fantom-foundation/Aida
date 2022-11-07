package operation

import (
	"bytes"
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"io"
	"math/rand"
	"os"
	"testing"
	"time"
)

func initFinalise(t *testing.T) (*dict.DictionaryContext, *Finalise, bool) {
	rand.Seed(time.Now().UnixNano())
	deleteEmpty := rand.Intn(2) == 1
	// create dictionary context
	dict := dict.NewDictionaryContext()

	// create new operation
	op := NewFinalise(deleteEmpty)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != FinaliseID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, deleteEmpty
}

// TestFinaliseReadWrite writes a new Finalise object into a buffer, reads from it,
// and checks equality.
func TestFinaliseReadWrite(t *testing.T) {
	_, op1, _ := initFinalise(t)
	testOperationReadWrite(t, op1, ReadFinalise)
}

// TestFinaliseDebug creates a new Finalise object and checks its Debug message.
func TestFinaliseDebug(t *testing.T) {
	dict, op, deleteEmpty := initFinalise(t)

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
	label, f := operationLabels[FinaliseID]
	if !f {
		t.Fatalf("label for %d not found", FinaliseID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %t\n", label, deleteEmpty) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestFinaliseExecute
func TestFinaliseExecute(t *testing.T) {
	dict, op, deleteEmpty := initFinalise(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{FinaliseID, []any{deleteEmpty}}}
	mock.compareRecordings(expected, t)
}

package operation

import (
	"bytes"
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"io"
	"os"
	"testing"
)

func initEndTransaction(t *testing.T) (*dict.DictionaryContext, *EndTransaction) {
	// create dictionary context
	dict := dict.NewDictionaryContext()

	// create new operation
	op := NewEndTransaction()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != EndTransactionID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op
}

// TestEndTransactionReadWrite writes a new EndTransaction object into a buffer, reads from it,
// and checks equality.
func TestEndTransactionReadWrite(t *testing.T) {
	_, op1 := initEndTransaction(t)
	testOperationReadWrite(t, op1, ReadEndTransaction)
}

// TestEndTransactionDebug creates a new EndTransaction object and checks its Debug message.
func TestEndTransactionDebug(t *testing.T) {
	dict, op := initEndTransaction(t)

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
	label, f := operationLabels[EndTransactionID]
	if !f {
		t.Fatalf("label for %d not found", EndTransactionID)
	}

	if buf.String() != fmt.Sprintf("\t%s\n", label) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestEndTransactionExecute
func TestEndTransactionExecute(t *testing.T) {
	dict, op := initEndTransaction(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	// currently EndTransaction isn't recorded
	//expected := []Record{{EndTransactionID, []any{}}}
	//mock.compareRecordings(expected, t)
	mock.compareRecordings([]Record{}, t)
}

package operation

import (
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
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
	testOperationDebug(t, dict, op, "")
}

// TestEndTransactionExecute
func TestEndTransactionExecute(t *testing.T) {
	dict, op := initEndTransaction(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{EndTransactionID, []any{}}}
	mock.compareRecordings(expected, t)
}

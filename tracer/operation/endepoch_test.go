package operation

import (
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

func initEndEpoch(t *testing.T) (*dict.DictionaryContext, *EndEpoch) {
	// create dictionary context
	dict := dict.NewDictionaryContext()

	// create new operation
	op := NewEndEpoch()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != EndEpochID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op
}

// TestEndEpochReadWrite writes a new EndEpoch object into a buffer, reads from it,
// and checks equality.
func TestEndEpochReadWrite(t *testing.T) {
	_, op1 := initEndEpoch(t)
	testOperationReadWrite(t, op1, ReadEndEpoch)
}

// TestEndEpochDebug creates a new EndEpoch object and checks its Debug message.
func TestEndEpochDebug(t *testing.T) {
	dict, op := initEndEpoch(t)
	testOperationDebug(t, dict, op, "")
}

// TestEndEpochExecute
func TestEndEpochExecute(t *testing.T) {
	dict, op := initEndEpoch(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{EndEpochID, []any{}}}
	mock.compareRecordings(expected, t)
}

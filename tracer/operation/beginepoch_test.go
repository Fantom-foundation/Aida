package operation

import (
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/dict"
)

func initBeginEpoch(t *testing.T) (*dict.DictionaryContext, *BeginEpoch) {
	// create dictionary context
	dict := dict.NewDictionaryContext()

	// create new operation
	op := NewBeginEpoch()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != BeginEpochID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op
}

// TestBeginEpochReadWrite writes a new BeginEpoch object into a buffer, reads from it,
// and checks equality.
func TestBeginEpochReadWrite(t *testing.T) {
	_, op1 := initBeginEpoch(t)
	testOperationReadWrite(t, op1, ReadBeginEpoch)
}

// TestBeginEpochDebug creates a new BeginEpoch object and checks its Debug message.
func TestBeginEpochDebug(t *testing.T) {
	dict, op := initBeginEpoch(t)
	testOperationDebug(t, dict, op, "")
}

// TestBeginEpochExecute
func TestBeginEpochExecute(t *testing.T) {
	dict, op := initBeginEpoch(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{BeginEpochID, []any{}}}
	mock.compareRecordings(expected, t)
}

package operation

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

func initBeginEpoch(t *testing.T) (*dict.DictionaryContext, *BeginEpoch) {
	rand.Seed(time.Now().UnixNano())
	num := rand.Uint64()

	// create dictionary context
	dict := dict.NewDictionaryContext()

	// create new operation
	op := NewBeginEpoch(num)
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
	testOperationDebug(t, dict, op, fmt.Sprintf("%v", op.EpochNumber))
}

// TestBeginEpochExecute
func TestBeginEpochExecute(t *testing.T) {
	dict, op := initBeginEpoch(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{BeginEpochID, []any{op.EpochNumber}}}
	mock.compareRecordings(expected, t)
}

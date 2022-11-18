package operation

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
)

func initEndEpoch(t *testing.T) (*dict.DictionaryContext, *EndEpoch) {
	rand.Seed(time.Now().UnixNano())
	num := rand.Uint64()

	// create dictionary context
	dict := dict.NewDictionaryContext()

	// create new operation
	op := NewEndEpoch(num)
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
	testOperationDebug(t, dict, op, fmt.Sprintf("%v", op.EpochNumber))
}

// TestEndEpochExecute
func TestEndEpochExecute(t *testing.T) {
	dict, op := initEndEpoch(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{EndEpochID, []any{op.EpochNumber}}}
	mock.compareRecordings(expected, t)
}

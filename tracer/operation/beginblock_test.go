package operation

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
)

func initBeginBlock(t *testing.T) (*dictionary.DictionaryContext, *BeginBlock, uint64) {
	rand.Seed(time.Now().UnixNano())
	blId := rand.Uint64()

	// create dictionary context
	dict := dictionary.NewDictionaryContext()

	// create new operation
	op := NewBeginBlock(blId)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != BeginBlockID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, blId
}

// TestBeginBlockReadWrite writes a new BeginBlock object into a buffer, reads from it,
// and checks equality.
func TestBeginBlockReadWrite(t *testing.T) {
	_, op1, _ := initBeginBlock(t)
	testOperationReadWrite(t, op1, ReadBeginBlock)
}

// TestBeginBlockDebug creates a new BeginBlock object and checks its Debug message.
func TestBeginBlockDebug(t *testing.T) {
	dict, op, value := initBeginBlock(t)
	testOperationDebug(t, dict, op, fmt.Sprint(value))
}

// TestBeginBlockExecute
func TestBeginBlockExecute(t *testing.T) {
	dict, op, _ := initBeginBlock(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{BeginBlockID, []any{op.BlockNumber}}}
	mock.compareRecordings(expected, t)
}

package operation

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
)

func initEndBlock(t *testing.T) (*dict.DictionaryContext, *EndBlock) {
	rand.Seed(time.Now().UnixNano())
	blk := rand.Uint64()

	// create dictionary context
	dict := dict.NewDictionaryContext()

	// create new operation
	op := NewEndBlock(blk)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != EndBlockID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op
}

// TestEndBlockReadWrite writes a new EndBlock object into a buffer, reads from it,
// and checks equality.
func TestEndBlockReadWrite(t *testing.T) {
	_, op1 := initEndBlock(t)
	testOperationReadWrite(t, op1, ReadEndBlock)
}

// TestEndBlockDebug creates a new EndBlock object and checks its Debug message.
func TestEndBlockDebug(t *testing.T) {
	dict, op := initEndBlock(t)
	testOperationDebug(t, dict, op, fmt.Sprintf("%v", op.BlockNumber))
}

// TestEndBlockExecute
func TestEndBlockExecute(t *testing.T) {
	dict, op := initEndBlock(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{EndBlockID, []any{op.BlockNumber}}}
	mock.compareRecordings(expected, t)
}

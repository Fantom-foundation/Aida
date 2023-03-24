package operation

import (
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

func initEndBlock(t *testing.T) (*context.Context, *EndBlock) {
	// create context context
	dict := context.NewContext()

	// create new operation
	op := NewEndBlock()
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
	testOperationDebug(t, dict, op, "")
}

// TestEndBlockExecute
func TestEndBlockExecute(t *testing.T) {
	dict, op := initEndBlock(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{EndBlockID, []any{}}}
	mock.compareRecordings(expected, t)
}

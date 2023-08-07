package operation

import (
	"testing"

	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/context"
)

func initEndBlock(t *testing.T) (*context.Replay, *EndBlock) {
	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewEndBlock()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != EndBlockID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op
}

// TestEndBlockReadWrite writes a new EndBlock object into a buffer, reads from it,
// and checks equality.
func TestEndBlockReadWrite(t *testing.T) {
	_, op1 := initEndBlock(t)
	testOperationReadWrite(t, op1, ReadEndBlock)
}

// TestEndBlockDebug creates a new EndBlock object and checks its Debug message.
func TestEndBlockDebug(t *testing.T) {
	ctx, op := initEndBlock(t)
	testOperationDebug(t, ctx, op, "")
}

// TestEndBlockExecute
func TestEndBlockExecute(t *testing.T) {
	ctx, op := initEndBlock(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{EndBlockID, []any{}}}
	mock.compareRecordings(expected, t)
}

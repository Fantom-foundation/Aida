package operation

import (
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

func initEndSyncPeriod(t *testing.T) (*context.Context, *EndSyncPeriod) {
	// create context context
	dict := context.NewContext()

	// create new operation
	op := NewEndSyncPeriod()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != EndSyncPeriodID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op
}

// TestEndSyncPeriodReadWrite writes a new EndSyncPeriod object into a buffer, reads from it,
// and checks equality.
func TestEndSyncPeriodReadWrite(t *testing.T) {
	_, op1 := initEndSyncPeriod(t)
	testOperationReadWrite(t, op1, ReadEndSyncPeriod)
}

// TestEndSyncPeriodDebug creates a new EndSyncPeriod object and checks its Debug message.
func TestEndSyncPeriodDebug(t *testing.T) {
	dict, op := initEndSyncPeriod(t)
	testOperationDebug(t, dict, op, "")
}

// TestEndSyncPeriodExecute
func TestEndSyncPeriodExecute(t *testing.T) {
	dict, op := initEndSyncPeriod(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{EndSyncPeriodID, []any{}}}
	mock.compareRecordings(expected, t)
}

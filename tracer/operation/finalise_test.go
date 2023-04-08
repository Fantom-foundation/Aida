package operation

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

func initFinalise(t *testing.T) (*context.Replay, *Finalise, bool) {
	rand.Seed(time.Now().UnixNano())
	deleteEmpty := rand.Intn(2) == 1
	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewFinalise(deleteEmpty)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != FinaliseID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, deleteEmpty
}

// TestFinaliseReadWrite writes a new Finalise object into a buffer, reads from it,
// and checks equality.
func TestFinaliseReadWrite(t *testing.T) {
	_, op1, _ := initFinalise(t)
	testOperationReadWrite(t, op1, ReadFinalise)
}

// TestFinaliseDebug creates a new Finalise object and checks its Debug message.
func TestFinaliseDebug(t *testing.T) {
	ctx, op, deleteEmpty := initFinalise(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(deleteEmpty))
}

// TestFinaliseExecute
func TestFinaliseExecute(t *testing.T) {
	ctx, op, deleteEmpty := initFinalise(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{FinaliseID, []any{deleteEmpty}}}
	mock.compareRecordings(expected, t)
}

package operation

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/context"
)

func initBeginSyncPeriod(t *testing.T) (*context.Replay, *BeginSyncPeriod) {
	rand.Seed(time.Now().UnixNano())
	num := rand.Uint64()

	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewBeginSyncPeriod(num)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != BeginSyncPeriodID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op
}

// TestBeginSyncPeriodReadWrite writes a new BeginSyncPeriod object into a buffer, reads from it,
// and checks equality.
func TestBeginSyncPeriodReadWrite(t *testing.T) {
	_, op1 := initBeginSyncPeriod(t)
	testOperationReadWrite(t, op1, ReadBeginSyncPeriod)
}

// TestBeginSyncPeriodDebug creates a new BeginSyncPeriod object and checks its Debug message.
func TestBeginSyncPeriodDebug(t *testing.T) {
	ctx, op := initBeginSyncPeriod(t)
	testOperationDebug(t, ctx, op, fmt.Sprintf("%v", op.SyncPeriodNumber))
}

// TestBeginSyncPeriodExecute
func TestBeginSyncPeriodExecute(t *testing.T) {
	ctx, op := initBeginSyncPeriod(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{BeginSyncPeriodID, []any{op.SyncPeriodNumber}}}
	mock.compareRecordings(expected, t)
}

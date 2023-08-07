package operation

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/context"
)

func initBeginTransaction(t *testing.T) (*context.Replay, *BeginTransaction) {
	rand.Seed(time.Now().UnixNano())
	num := rand.Uint32()

	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewBeginTransaction(num)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != BeginTransactionID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op
}

// TestBeginTransactionReadWrite writes a new BeginTransaction object into a buffer, reads from it,
// and checks equality.
func TestBeginTransactionReadWrite(t *testing.T) {
	_, op1 := initBeginTransaction(t)
	testOperationReadWrite(t, op1, ReadBeginTransaction)
}

// TestBeginTransactionDebug creates a new BeginTransaction object and checks its Debug message.
func TestBeginTransactionDebug(t *testing.T) {
	ctx, op := initBeginTransaction(t)
	testOperationDebug(t, ctx, op, fmt.Sprintf("%v", op.TransactionNumber))
}

// TestBeginTransactionExecute
func TestBeginTransactionExecute(t *testing.T) {
	ctx, op := initBeginTransaction(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{BeginTransactionID, []any{op.TransactionNumber}}}
	mock.compareRecordings(expected, t)
}

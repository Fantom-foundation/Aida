package operation

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/context"
)

func initBeginBlock(t *testing.T) (*context.Replay, *BeginBlock, uint64) {
	rand.Seed(time.Now().UnixNano())
	blId := rand.Uint64()

	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewBeginBlock(blId)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != BeginBlockID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, blId
}

// TestBeginBlockReadWrite writes a new BeginBlock object into a buffer, reads from it,
// and checks equality.
func TestBeginBlockReadWrite(t *testing.T) {
	_, op1, _ := initBeginBlock(t)
	testOperationReadWrite(t, op1, ReadBeginBlock)
}

// TestBeginBlockDebug creates a new BeginBlock object and checks its Debug message.
func TestBeginBlockDebug(t *testing.T) {
	ctx, op, value := initBeginBlock(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(value))
}

// TestBeginBlockExecute
func TestBeginBlockExecute(t *testing.T) {
	ctx, op, _ := initBeginBlock(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{BeginBlockID, []any{op.BlockNumber}}}
	mock.compareRecordings(expected, t)
}

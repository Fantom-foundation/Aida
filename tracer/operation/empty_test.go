package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initEmpty(t *testing.T) (*context.Replay, *Empty, common.Address) {
	addr := getRandomAddress(t)
	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)

	// create new operation
	op := NewEmpty(contract)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != EmptyID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, addr
}

// TestEmptyReadWrite writes a new Empty object into a buffer, reads from it,
// and checks equality.
func TestEmptyReadWrite(t *testing.T) {
	_, op1, _ := initEmpty(t)
	testOperationReadWrite(t, op1, ReadEmpty)
}

// TestEmptyDebug creates a new Empty object and checks its Debug message.
func TestEmptyDebug(t *testing.T) {
	ctx, op, addr := initEmpty(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr))
}

// TestEmptyExecute
func TestEmptyExecute(t *testing.T) {
	ctx, op, addr := initEmpty(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{EmptyID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

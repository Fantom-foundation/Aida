package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initSuicide(t *testing.T) (*context.Replay, *Suicide, common.Address) {
	addr := getRandomAddress(t)
	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)

	// create new operation
	op := NewSuicide(contract)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SuicideID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, addr
}

// TestSuicideReadWrite writes a new Suicide object into a buffer, reads from it,
// and checks equality.
func TestSuicideReadWrite(t *testing.T) {
	_, op1, _ := initSuicide(t)
	testOperationReadWrite(t, op1, ReadSuicide)
}

// TestSuicideDebug creates a new Suicide object and checks its Debug message.
func TestSuicideDebug(t *testing.T) {
	ctx, op, addr := initSuicide(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr))
}

// TestSuicideExecute
func TestSuicideExecute(t *testing.T) {
	ctx, op, addr := initSuicide(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{SuicideID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initGetCodeHash(t *testing.T) (*context.Replay, *GetCodeHash, common.Address) {
	addr := getRandomAddress(t)
	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)

	// create new operation
	op := NewGetCodeHash(contract)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCodeHashID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, addr
}

// TestGetCodeHashReadWrite writes a new GetCodeHash object into a buffer, reads from it,
// and checks equality.
func TestGetCodeHashReadWrite(t *testing.T) {
	_, op1, _ := initGetCodeHash(t)
	testOperationReadWrite(t, op1, ReadGetCodeHash)
}

// TestGetCodeHashDebug creates a new GetCodeHash object and checks its Debug message.
func TestGetCodeHashDebug(t *testing.T) {
	ctx, op, addr := initGetCodeHash(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr))
}

// TestGetCodeHashExecute
func TestGetCodeHashExecute(t *testing.T) {
	ctx, op, addr := initGetCodeHash(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{GetCodeHashID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

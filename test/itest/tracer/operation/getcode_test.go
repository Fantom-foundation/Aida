package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initGetCode(t *testing.T) (*context.Replay, *GetCode, common.Address) {
	addr := getRandomAddress(t)
	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)

	// create new operation
	op := NewGetCode(contract)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCodeID {
		t.Fatalf("wrong ID returned")
	}
	return ctx, op, addr
}

// TestGetCodeReadWrite writes a new GetCode object into a buffer, reads from it,
// and checks equality.
func TestGetCodeReadWrite(t *testing.T) {
	_, op1, _ := initGetCode(t)
	testOperationReadWrite(t, op1, ReadGetCode)
}

// TestGetCodeDebug creates a new GetCode object and checks its Debug message.
func TestGetCodeDebug(t *testing.T) {
	ctx, op, addr := initGetCode(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr))
}

// TestGetCodeExecute creates a new GetCode object and checks its execution signature.
func TestGetCodeExecute(t *testing.T) {
	ctx, op, addr := initGetCode(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{GetCodeID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

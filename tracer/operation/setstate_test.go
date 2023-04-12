package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initSetState(t *testing.T) (*context.Replay, *SetState, common.Address, common.Hash, common.Hash) {
	addr := getRandomAddress(t)
	storage := getRandomAddress(t).Hash()
	value := getRandomAddress(t).Hash()

	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)
	sIdx, _ := ctx.EncodeKey(storage)

	// create new operation
	op := NewSetState(contract, sIdx, value)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SetStateID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, addr, storage, value
}

// TestSetStateReadWrite writes a new SetState object into a buffer, reads from it,
// and checks equality.
func TestSetStateReadWrite(t *testing.T) {
	_, op1, _, _, _ := initSetState(t)
	testOperationReadWrite(t, op1, ReadSetState)
}

// TestSetStateDebug creates a new SetState object and checks its Debug message.
func TestSetStateDebug(t *testing.T) {
	ctx, op, addr, storage, value := initSetState(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr, storage, value))
}

// TestSetStateExecute
func TestSetStateExecute(t *testing.T) {
	ctx, op, addr, storage, value := initSetState(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{SetStateID, []any{addr, storage, value}}}
	mock.compareRecordings(expected, t)
}

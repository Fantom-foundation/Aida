package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initGetCommittedState(t *testing.T) (*context.Replay, *GetCommittedState, common.Address, common.Hash) {
	addr := getRandomAddress(t)
	storage := getRandomAddress(t).Hash()

	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)
	sIdx, _ := ctx.EncodeKey(storage)

	// create new operation
	op := NewGetCommittedState(contract, sIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCommittedStateID {
		t.Fatalf("wrong ID returned")
	}
	return ctx, op, addr, storage
}

// TestGetCommittedStateReadWrite writes a new GetCommittedState object into a buffer, reads from it,
// and checks equality.
func TestGetCommittedStateReadWrite(t *testing.T) {
	_, op1, _, _ := initGetCommittedState(t)
	testOperationReadWrite(t, op1, ReadGetCommittedState)
}

// TestGetCommittedStateDebug creates a new GetCommittedState object and checks its Debug message.
func TestGetCommittedStateDebug(t *testing.T) {
	ctx, op, addr, storage := initGetCommittedState(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr, storage))
}

// TestGetCommittedStateExecute
func TestGetCommittedStateExecute(t *testing.T) {
	ctx, op, addr, storage := initGetCommittedState(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{GetCommittedStateID, []any{addr, storage}}}
	mock.compareRecordings(expected, t)
}

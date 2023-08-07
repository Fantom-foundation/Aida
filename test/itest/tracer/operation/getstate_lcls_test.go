package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initGetStateLcls(t *testing.T) (*context.Replay, *GetStateLcls, common.Address, common.Hash) {
	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewGetStateLcls()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetStateLclsID {
		t.Fatalf("wrong ID returned")
	}

	addr := getRandomAddress(t)
	ctx.EncodeContract(addr)

	storage := getRandomAddress(t).Hash()
	ctx.EncodeKey(storage)

	return ctx, op, addr, storage
}

// TestGetStateLclsReadWrite writes a new GetStateLcls object into a buffer, reads from it,
// and checks equality.
func TestGetStateLclsReadWrite(t *testing.T) {
	_, op1, _, _ := initGetStateLcls(t)
	testOperationReadWrite(t, op1, ReadGetStateLcls)
}

// TestGetStateLclsDebug creates a new GetStateLcls object and checks its Debug message.
func TestGetStateLclsDebug(t *testing.T) {
	ctx, op, addr, storage := initGetStateLcls(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr, storage))
}

// TestGetStateLclsExecute
func TestGetStateLclsExecute(t *testing.T) {
	ctx, op, addr, storage := initGetStateLcls(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{GetStateID, []any{addr, storage}}}
	mock.compareRecordings(expected, t)
}

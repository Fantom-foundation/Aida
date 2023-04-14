package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initGetCommittedStateLcls(t *testing.T) (*context.Replay, *GetCommittedStateLcls, common.Address, common.Hash) {
	// create context context
	ctx := context.NewReplay()

	// create new operation
	op := NewGetCommittedStateLcls()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCommittedStateLclsID {
		t.Fatalf("wrong ID returned")
	}

	addr := getRandomAddress(t)
	ctx.EncodeContract(addr)

	storage := getRandomAddress(t).Hash()
	ctx.EncodeKey(storage)

	return ctx, op, addr, storage
}

// TestGetCommittedStateLclsReadWrite writes a new GetCommittedStateLcls object into a buffer, reads from it,
// and checks equality.
func TestGetCommittedStateLclsReadWrite(t *testing.T) {
	_, op1, _, _ := initGetCommittedStateLcls(t)
	testOperationReadWrite(t, op1, ReadGetCommittedStateLcls)
}

// TestGetCommittedStateLclsDebug creates a new GetCommittedStateLcls object and checks its Debug message.
func TestGetCommittedStateLclsDebug(t *testing.T) {
	ctx, op, addr, storage := initGetCommittedStateLcls(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr, storage))
}

// TestGetCommittedStateLclsExecute
func TestGetCommittedStateLclsExecute(t *testing.T) {
	ctx, op, addr, storage := initGetCommittedStateLcls(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{GetCommittedStateID, []any{addr, storage}}}
	mock.compareRecordings(expected, t)
}

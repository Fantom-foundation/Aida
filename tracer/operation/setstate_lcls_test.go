package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initSetStateLcls(t *testing.T) (*context.Replay, *SetStateLcls, common.Address, common.Hash, common.Hash) {
	value := getRandomAddress(t).Hash()

	// create new operation
	op := NewSetStateLcls(value)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SetStateLclsID {
		t.Fatalf("wrong ID returned")
	}

	// create context context
	ctx := context.NewReplay()

	addr := getRandomAddress(t)
	ctx.EncodeContract(addr)

	storage := getRandomAddress(t).Hash()
	ctx.EncodeKey(storage)

	return ctx, op, addr, storage, value
}

// TestSetStateLclsReadWrite writes a new SetStateLcls object into a buffer, reads from it,
// and checks equality.
func TestSetStateLclsReadWrite(t *testing.T) {
	_, op1, _, _, _ := initSetStateLcls(t)
	testOperationReadWrite(t, op1, ReadSetStateLcls)
}

// TestSetStateLclsDebug creates a new SetStateLcls object and checks its Debug message.
func TestSetStateLclsDebug(t *testing.T) {
	ctx, op, addr, storage, value := initSetStateLcls(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr, storage, value))
}

// TestSetStateLclsExecute
func TestSetStateLclsExecute(t *testing.T) {
	ctx, op, addr, storage, value := initSetStateLcls(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{SetStateID, []any{addr, storage, value}}}
	mock.compareRecordings(expected, t)
}

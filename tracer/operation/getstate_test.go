package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initGetState(t *testing.T) (*context.Context, *GetState, common.Address, common.Hash) {
	addr := getRandomAddress(t)
	storage := getRandomAddress(t).Hash()

	// create context context
	dict := context.NewContext()
	contract := dict.EncodeContract(addr)
	sIdx, _ := dict.EncodeKey(storage)

	// create new operation
	op := NewGetState(contract, sIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetStateID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr, storage
}

// TestGetStateReadWrite writes a new GetState object into a buffer, reads from it,
// and checks equality.
func TestGetStateReadWrite(t *testing.T) {
	_, op1, _, _ := initGetState(t)
	testOperationReadWrite(t, op1, ReadGetState)
}

// TestGetStateDebug creates a new GetState object and checks its Debug message.
func TestGetStateDebug(t *testing.T) {
	dict, op, addr, storage := initGetState(t)
	testOperationDebug(t, dict, op, fmt.Sprint(addr, storage))
}

// TestGetStateExecute
func TestGetStateExecute(t *testing.T) {
	dict, op, addr, storage := initGetState(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetStateID, []any{addr, storage}}}
	mock.compareRecordings(expected, t)
}

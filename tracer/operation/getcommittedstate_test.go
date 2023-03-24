package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initGetCommittedState(t *testing.T) (*context.Context, *GetCommittedState, common.Address, common.Hash) {
	addr := getRandomAddress(t)
	storage := getRandomAddress(t).Hash()

	// create context context
	dict := context.NewContext()
	contract := dict.EncodeContract(addr)
	sIdx, _ := dict.EncodeKey(storage)

	// create new operation
	op := NewGetCommittedState(contract, sIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCommittedStateID {
		t.Fatalf("wrong ID returned")
	}
	return dict, op, addr, storage
}

// TestGetCommittedStateReadWrite writes a new GetCommittedState object into a buffer, reads from it,
// and checks equality.
func TestGetCommittedStateReadWrite(t *testing.T) {
	_, op1, _, _ := initGetCommittedState(t)
	testOperationReadWrite(t, op1, ReadGetCommittedState)
}

// TestGetCommittedStateDebug creates a new GetCommittedState object and checks its Debug message.
func TestGetCommittedStateDebug(t *testing.T) {
	dict, op, addr, storage := initGetCommittedState(t)
	testOperationDebug(t, dict, op, fmt.Sprint(addr, storage))
}

// TestGetCommittedStateExecute
func TestGetCommittedStateExecute(t *testing.T) {
	dict, op, addr, storage := initGetCommittedState(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetCommittedStateID, []any{addr, storage}}}
	mock.compareRecordings(expected, t)
}

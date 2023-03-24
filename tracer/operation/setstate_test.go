package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initSetState(t *testing.T) (*context.Context, *SetState, common.Address, common.Hash, common.Hash) {
	addr := getRandomAddress(t)
	storage := getRandomAddress(t).Hash()
	value := getRandomAddress(t).Hash()

	// create context context
	dict := context.NewContext()
	contract := dict.EncodeContract(addr)
	sIdx, _ := dict.EncodeKey(storage)

	// create new operation
	op := NewSetState(contract, sIdx, value)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SetStateID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr, storage, value
}

// TestSetStateReadWrite writes a new SetState object into a buffer, reads from it,
// and checks equality.
func TestSetStateReadWrite(t *testing.T) {
	_, op1, _, _, _ := initSetState(t)
	testOperationReadWrite(t, op1, ReadSetState)
}

// TestSetStateDebug creates a new SetState object and checks its Debug message.
func TestSetStateDebug(t *testing.T) {
	dict, op, addr, storage, value := initSetState(t)
	testOperationDebug(t, dict, op, fmt.Sprint(addr, storage, value))
}

// TestSetStateExecute
func TestSetStateExecute(t *testing.T) {
	dict, op, addr, storage, value := initSetState(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{SetStateID, []any{addr, storage, value}}}
	mock.compareRecordings(expected, t)
}

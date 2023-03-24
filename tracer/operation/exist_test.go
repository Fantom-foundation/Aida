package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initExist(t *testing.T) (*context.Context, *Exist, common.Address) {
	addr := getRandomAddress(t)
	// create context context
	dict := context.NewContext()
	contract := dict.EncodeContract(addr)

	// create new operation
	op := NewExist(contract)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != ExistID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr
}

// TestExistReadWrite writes a new Exist object into a buffer, reads from it,
// and checks equality.
func TestExistReadWrite(t *testing.T) {
	_, op1, _ := initExist(t)
	testOperationReadWrite(t, op1, ReadExist)
}

// TestExistDebug creates a new Exist object and checks its Debug message.
func TestExistDebug(t *testing.T) {
	dict, op, addr := initExist(t)
	testOperationDebug(t, dict, op, fmt.Sprint(addr))
}

// TestExistExecute
func TestExistExecute(t *testing.T) {
	dict, op, addr := initExist(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{ExistID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

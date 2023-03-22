package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
	"github.com/ethereum/go-ethereum/common"
)

func initSuicide(t *testing.T) (*dictionary.Context, *Suicide, common.Address) {
	addr := getRandomAddress(t)
	// create dictionary context
	dict := dictionary.NewContext()
	contract := dict.EncodeContract(addr)

	// create new operation
	op := NewSuicide(contract)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SuicideID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr
}

// TestSuicideReadWrite writes a new Suicide object into a buffer, reads from it,
// and checks equality.
func TestSuicideReadWrite(t *testing.T) {
	_, op1, _ := initSuicide(t)
	testOperationReadWrite(t, op1, ReadSuicide)
}

// TestSuicideDebug creates a new Suicide object and checks its Debug message.
func TestSuicideDebug(t *testing.T) {
	dict, op, addr := initSuicide(t)
	testOperationDebug(t, dict, op, fmt.Sprint(addr))
}

// TestSuicideExecute
func TestSuicideExecute(t *testing.T) {
	dict, op, addr := initSuicide(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{SuicideID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

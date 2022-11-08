package operation

import (
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func initEmpty(t *testing.T) (*dict.DictionaryContext, *Empty, common.Address) {
	addr := getRandomAddress(t)
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)

	// create new operation
	op := NewEmpty(cIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != EmptyID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr
}

// TestEmptyReadWrite writes a new Empty object into a buffer, reads from it,
// and checks equality.
func TestEmptyReadWrite(t *testing.T) {
	_, op1, _ := initEmpty(t)
	testOperationReadWrite(t, op1, ReadEmpty)
}

// TestEmptyDebug creates a new Empty object and checks its Debug message.
func TestEmptyDebug(t *testing.T) {
	dict, op, addr := initEmpty(t)
	testOperationDebug(t, dict, op, fmt.Sprint(addr))
}

// TestEmptyExecute
func TestEmptyExecute(t *testing.T) {
	dict, op, addr := initEmpty(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{EmptyID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

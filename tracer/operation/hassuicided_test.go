package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
	"github.com/ethereum/go-ethereum/common"
)

func initHasSuicided(t *testing.T) (*dictionary.DictionaryContext, *HasSuicided, common.Address) {
	addr := getRandomAddress(t)
	// create dictionary context
	dict := dictionary.NewDictionaryContext()
	cIdx := dictionary.EncodeContract(addr)

	// create new operation
	op := NewHasSuicided(cIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != HasSuicidedID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr
}

// TestHasSuicidedReadWrite writes a new HasSuicided object into a buffer, reads from it,
// and checks equality.
func TestHasSuicidedReadWrite(t *testing.T) {
	_, op1, _ := initHasSuicided(t)
	testOperationReadWrite(t, op1, ReadHasSuicided)
}

// TestHasSuicidedDebug creates a new HasSuicided object and checks its Debug message.
func TestHasSuicidedDebug(t *testing.T) {
	dict, op, addr := initHasSuicided(t)
	testOperationDebug(t, dict, op, fmt.Sprint(addr))
}

// TestHasSuicidedExecute
func TestHasSuicidedExecute(t *testing.T) {
	dict, op, addr := initHasSuicided(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{HasSuicidedID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

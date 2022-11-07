package operation

import (
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func initHasSuicided(t *testing.T) (*dict.DictionaryContext, *HasSuicided, common.Address) {
	addr := getRandomAddress(t)
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)

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
	testOperationDebug(t, dict, op, HasSuicidedID, func(label string) string {
		return fmt.Sprintf("\t%s: %s\n", label, addr)
	})
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

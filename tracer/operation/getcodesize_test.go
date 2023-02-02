package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/ethereum/go-ethereum/common"
)

func initGetCodeSize(t *testing.T) (*dict.DictionaryContext, *GetCodeSize, common.Address) {
	addr := getRandomAddress(t)
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)

	// create new operation
	op := NewGetCodeSize(cIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCodeSizeID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr
}

// TestGetCodeSizeReadWrite writes a new GetCodeSize object into a buffer, reads from it,
// and checks equality.
func TestGetCodeSizeReadWrite(t *testing.T) {
	_, op1, _ := initGetCodeSize(t)
	testOperationReadWrite(t, op1, ReadGetCodeSize)
}

// TestGetCodeSizeDebug creates a new GetCodeSize object and checks its Debug message.
func TestGetCodeSizeDebug(t *testing.T) {
	dict, op, addr := initGetCodeSize(t)
	testOperationDebug(t, dict, op, fmt.Sprint(addr))
}

// TestGetCodeSizeExecute creates a new GetCodeSize object and checks its execution signature.
func TestGetCodeSizeExecute(t *testing.T) {
	dict, op, addr := initGetCodeSize(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetCodeSizeID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

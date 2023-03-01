package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
	"github.com/ethereum/go-ethereum/common"
)

func initGetCode(t *testing.T) (*dict.DictionaryContext, *GetCode, common.Address) {
	addr := getRandomAddress(t)
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)

	// create new operation
	op := NewGetCode(cIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCodeID {
		t.Fatalf("wrong ID returned")
	}
	return dict, op, addr
}

// TestGetCodeReadWrite writes a new GetCode object into a buffer, reads from it,
// and checks equality.
func TestGetCodeReadWrite(t *testing.T) {
	_, op1, _ := initGetCode(t)
	testOperationReadWrite(t, op1, ReadGetCode)
}

// TestGetCodeDebug creates a new GetCode object and checks its Debug message.
func TestGetCodeDebug(t *testing.T) {
	dict, op, addr := initGetCode(t)
	testOperationDebug(t, dict, op, fmt.Sprint(addr))
}

// TestGetCodeExecute creates a new GetCode object and checks its execution signature.
func TestGetCodeExecute(t *testing.T) {
	dict, op, addr := initGetCode(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetCodeID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

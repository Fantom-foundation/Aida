package operation

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
	"github.com/ethereum/go-ethereum/common"
)

func initSetCode(t *testing.T) (*dictionary.Context, *SetCode, common.Address, []byte) {
	rand.Seed(time.Now().UnixNano())
	addr := getRandomAddress(t)
	code := make([]byte, 100)
	rand.Read(code)

	// create dictionary context
	dict := dictionary.NewContext()
	contract := dict.EncodeContract(addr)
	bcontract := dict.EncodeCode(code)

	// create new operation
	op := NewSetCode(contract, bcontract)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SetCodeID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr, code
}

// TestSetCodeReadWrite writes a new SetCode object into a buffer, reads from it,
// and checks equality.
func TestSetCodeReadWrite(t *testing.T) {
	_, op1, _, _ := initSetCode(t)
	testOperationReadWrite(t, op1, ReadSetCode)
}

// TestSetCodeDebug creates a new SetCode object and checks its Debug message.
func TestSetCodeDebug(t *testing.T) {
	dict, op, addr, value := initSetCode(t)
	testOperationDebug(t, dict, op, fmt.Sprint(addr, value))
}

// TestSetCodeExecute
func TestSetCodeExecute(t *testing.T) {
	dict, op, addr, code := initSetCode(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{SetCodeID, []any{addr, code}}}
	mock.compareRecordings(expected, t)
}

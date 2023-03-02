package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
	"github.com/ethereum/go-ethereum/common"
)

func initGetNonce(t *testing.T) (*dictionary.Context, *GetNonce, common.Address) {
	addr := getRandomAddress(t)
	// create dictionary context
	dict := dictionary.NewContext()
	cIdx := dict.EncodeContract(addr)

	// create new operation
	op := NewGetNonce(cIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetNonceID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr
}

// TestGetNonceReadWrite writes a new GetNonce object into a buffer, reads from it,
// and checks equality.
func TestGetNonceReadWrite(t *testing.T) {
	_, op1, _ := initGetNonce(t)
	testOperationReadWrite(t, op1, ReadGetNonce)
}

// TestGetNonceDebug creates a new GetNonce object and checks its Debug message.
func TestGetNonceDebug(t *testing.T) {
	dict, op, addr := initGetNonce(t)
	testOperationDebug(t, dict, op, fmt.Sprint(addr))
}

// TestGetNonceExecute
func TestGetNonceExecute(t *testing.T) {
	dict, op, addr := initGetNonce(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetNonceID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

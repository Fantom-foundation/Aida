package operation

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
	"github.com/ethereum/go-ethereum/common"
)

func initSetNonce(t *testing.T) (*dictionary.DictionaryContext, *SetNonce, common.Address, uint64) {
	rand.Seed(time.Now().UnixNano())
	addr := getRandomAddress(t)
	nonce := rand.Uint64()

	// create dictionary context
	dict := dictionary.NewDictionaryContext()
	cIdx := dictionary.EncodeContract(addr)

	// create new operation
	op := NewSetNonce(cIdx, nonce)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SetNonceID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr, nonce
}

// TestSetNonceReadWrite writes a new SetNonce object into a buffer, reads from it,
// and checks equality.
func TestSetNonceReadWrite(t *testing.T) {
	_, op1, _, _ := initSetNonce(t)
	testOperationReadWrite(t, op1, ReadSetNonce)
}

// TestSetNonceDebug creates a new SetNonce object and checks its Debug message.
func TestSetNonceDebug(t *testing.T) {
	dict, op, addr, value := initSetNonce(t)
	testOperationDebug(t, dict, op, fmt.Sprint(addr, value))
}

// TestSetNonceExecute
func TestSetNonceExecute(t *testing.T) {
	dict, op, addr, nonce := initSetNonce(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{SetNonceID, []any{addr, nonce}}}
	mock.compareRecordings(expected, t)
}

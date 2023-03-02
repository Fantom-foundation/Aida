package operation

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
	"github.com/ethereum/go-ethereum/common"
)

func initGetStateLccs(t *testing.T) (*dictionary.Context, *GetStateLccs, common.Address, common.Hash, common.Hash) {
	rand.Seed(time.Now().UnixNano())
	pos := 0

	// create dictionary context
	dict := dictionary.NewContext()

	// create new operation
	op := NewGetStateLccs(pos)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetStateLccsID {
		t.Fatalf("wrong ID returned")
	}

	addr := getRandomAddress(t)
	dictionary.EncodeContract(addr)

	storage := getRandomAddress(t).Hash()
	dictionary.EncodeStorage(storage)

	storage2 := getRandomAddress(t).Hash()

	return dict, op, addr, storage, storage2
}

// TestGetStateLccsReadWrite writes a new GetStateLccs object into a buffer, reads from it,
// and checks equality.
func TestGetStateLccsReadWrite(t *testing.T) {
	_, op1, _, _, _ := initGetStateLccs(t)
	testOperationReadWrite(t, op1, ReadGetStateLccs)
}

// TestGetStateLccsDebug creates a new GetStateLccs object and checks its Debug message.
func TestGetStateLccsDebug(t *testing.T) {
	dict, op, addr, storage, _ := initGetStateLccs(t)
	testOperationDebug(t, dict, op, fmt.Sprint(addr, storage))
}

// TestGetStateLccsExecute
func TestGetStateLccsExecute(t *testing.T) {
	dict, op, addr, storage, storage2 := initGetStateLccs(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	dictionary.EncodeStorage(storage2)

	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetStateID, []any{addr, storage}}, {GetStateID, []any{addr, storage2}}}
	mock.compareRecordings(expected, t)
}

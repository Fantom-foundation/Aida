package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/dictionary"
	"github.com/ethereum/go-ethereum/common"
)

func initGetCommittedStateLcls(t *testing.T) (*dictionary.DictionaryContext, *GetCommittedStateLcls, common.Address, common.Hash) {
	// create dictionary context
	dict := dictionary.NewDictionaryContext()

	// create new operation
	op := NewGetCommittedStateLcls()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCommittedStateLclsID {
		t.Fatalf("wrong ID returned")
	}

	addr := getRandomAddress(t)
	dictionary.EncodeContract(addr)

	storage := getRandomAddress(t).Hash()
	dictionary.EncodeStorage(storage)

	return dict, op, addr, storage
}

// TestGetCommittedStateLclsReadWrite writes a new GetCommittedStateLcls object into a buffer, reads from it,
// and checks equality.
func TestGetCommittedStateLclsReadWrite(t *testing.T) {
	_, op1, _, _ := initGetCommittedStateLcls(t)
	testOperationReadWrite(t, op1, ReadGetCommittedStateLcls)
}

// TestGetCommittedStateLclsDebug creates a new GetCommittedStateLcls object and checks its Debug message.
func TestGetCommittedStateLclsDebug(t *testing.T) {
	dict, op, addr, storage := initGetCommittedStateLcls(t)
	testOperationDebug(t, dict, op, fmt.Sprint(addr, storage))
}

// TestGetCommittedStateLclsExecute
func TestGetCommittedStateLclsExecute(t *testing.T) {
	dict, op, addr, storage := initGetCommittedStateLcls(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetCommittedStateID, []any{addr, storage}}}
	mock.compareRecordings(expected, t)
}

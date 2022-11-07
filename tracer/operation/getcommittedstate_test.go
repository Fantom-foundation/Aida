package operation

import (
	"bytes"
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/ethereum/go-ethereum/common"
	"io"
	"os"
	"testing"
)

func initGetCommittedState(t *testing.T) (*dict.DictionaryContext, *GetCommittedState, common.Address, common.Hash) {
	addr := getRandomAddress(t)
	storage := getRandomAddress(t).Hash()

	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)
	sIdx, _ := dict.EncodeStorage(storage)

	// create new operation
	op := NewGetCommittedState(cIdx, sIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCommittedStateID {
		t.Fatalf("wrong ID returned")
	}
	return dict, op, addr, storage
}

// TestGetCommittedStateReadWrite writes a new GetCommittedState object into a buffer, reads from it,
// and checks equality.
func TestGetCommittedStateReadWrite(t *testing.T) {
	_, op1, _, _ := initGetCommittedState(t)
	testOperationReadWrite(t, op1, ReadGetCommittedState)
}

// TestGetCommittedStateDebug creates a new GetCommittedState object and checks its Debug message.
func TestGetCommittedStateDebug(t *testing.T) {
	dict, op, addr, storage := initGetCommittedState(t)

	// divert stdout to a buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// print debug message
	op.Debug(dict)

	// restore stdout
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// check debug message
	label, f := operationLabels[GetCommittedStateID]
	if !f {
		t.Fatalf("label for %d not found", GetCommittedStateID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s, %s\n", label, addr, storage) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestGetCommittedStateExecute
func TestGetCommittedStateExecute(t *testing.T) {
	dict, op, addr, storage := initGetCommittedState(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetCommittedStateID, []any{addr, storage}}}
	mock.compareRecordings(expected, t)
}

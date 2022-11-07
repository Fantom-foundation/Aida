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

func initGetStateLcls(t *testing.T) (*dict.DictionaryContext, *GetStateLcls, common.Address, common.Hash) {
	// create dictionary context
	dict := dict.NewDictionaryContext()

	// create new operation
	op := NewGetStateLcls()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetStateLclsID {
		t.Fatalf("wrong ID returned")
	}

	addr := getRandomAddress(t)
	dict.EncodeContract(addr)

	storage := getRandomAddress(t).Hash()
	dict.EncodeStorage(storage)

	return dict, op, addr, storage
}

// TestGetStateLclsReadWrite writes a new GetStateLcls object into a buffer, reads from it,
// and checks equality.
func TestGetStateLclsReadWrite(t *testing.T) {
	_, op1, _, _ := initGetStateLcls(t)
	testOperationReadWrite(t, op1, ReadGetStateLcls)
}

// TestGetStateLclsDebug creates a new GetStateLcls object and checks its Debug message.
func TestGetStateLclsDebug(t *testing.T) {
	dict, op, addr, storage := initGetStateLcls(t)

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
	label, f := operationLabels[GetStateLclsID]
	if !f {
		t.Fatalf("label for %d not found", GetStateLclsID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s, %s\n", label, addr, storage) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestGetStateLclsExecute
func TestGetStateLclsExecute(t *testing.T) {
	dict, op, addr, storage := initGetStateLcls(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetStateID, []any{addr, storage}}}
	mock.compareRecordings(expected, t)
}

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

func initGetNonce(t *testing.T) (*dict.DictionaryContext, *GetNonce, common.Address) {
	addr := getRandomAddress(t)
	// create dictionary context
	dict := dict.NewDictionaryContext()
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
	label, f := operationLabels[GetNonceID]
	if !f {
		t.Fatalf("label for %d not found", GetNonceID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s\n", label, addr) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
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

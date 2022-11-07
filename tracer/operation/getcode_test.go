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
	label, f := operationLabels[GetCodeID]
	if !f {
		t.Fatalf("label for %d not found", GetCodeID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s\n", label, addr) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
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

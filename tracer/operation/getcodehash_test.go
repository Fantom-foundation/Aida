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

func initGetCodeHash(t *testing.T) (*dict.DictionaryContext, *GetCodeHash, common.Address) {
	addr := getRandomAddress(t)
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)

	// create new operation
	op := NewGetCodeHash(cIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCodeHashID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr
}

// TestGetCodeHashReadWrite writes a new GetCodeHash object into a buffer, reads from it,
// and checks equality.
func TestGetCodeHashReadWrite(t *testing.T) {
	_, op1, _ := initGetCodeHash(t)
	testOperationReadWrite(t, op1, ReadGetCodeHash)
}

// TestGetCodeHashDebug creates a new GetCodeHash object and checks its Debug message.
func TestGetCodeHashDebug(t *testing.T) {
	dict, op, addr := initGetCodeHash(t)

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
	label, f := operationLabels[GetCodeHashID]
	if !f {
		t.Fatalf("label for %d not found", GetCodeHashID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s\n", label, addr) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestGetCodeHashExecute
func TestGetCodeHashExecute(t *testing.T) {
	dict, op, addr := initGetCodeHash(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetCodeHashID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

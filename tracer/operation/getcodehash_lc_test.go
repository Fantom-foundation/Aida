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

func initGetCodeHashLc(t *testing.T) (*dict.DictionaryContext, *GetCodeHashLc, common.Address) {
	// create dictionary context
	dict := dict.NewDictionaryContext()

	addr := getRandomAddress(t)
	dict.EncodeContract(addr)

	// create new operation
	op := NewGetCodeHashLc()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCodeHashLcID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr
}

// TestGetCodeHashLcReadWrite writes a new GetCodeHashLc object into a buffer, reads from it,
// and checks equality.
func TestGetCodeHashLcReadWrite(t *testing.T) {
	_, op1, _ := initGetCodeHashLc(t)
	testOperationReadWrite(t, op1, ReadGetCodeHashLc)
}

// TestGetCodeHashLcDebug creates a new GetCodeHashLc object and checks its Debug message.
func TestGetCodeHashLcDebug(t *testing.T) {
	dict, op, addr := initGetCodeHashLc(t)

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
	label, f := operationLabels[GetCodeHashLcID]
	if !f {
		t.Fatalf("label for %d not found", GetCodeHashLcID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s\n", label, addr) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestGetCodeHashLcExecute
func TestGetCodeHashLcExecute(t *testing.T) {
	dict, op, addr := initGetCodeHashLc(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetCodeHashID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

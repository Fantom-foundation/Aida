package operation

import (
	"bytes"
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/ethereum/go-ethereum/common"
	"io"
	"os"
	"reflect"
	"testing"
)

func initGetCodeSize(t *testing.T) (*dict.DictionaryContext, *GetCodeSize, common.Address) {
	addr := getRandomAddress(t)
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)

	// create new operation
	op := NewGetCodeSize(cIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCodeSizeID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr
}

// TestGetCodeSizeReadWrite writes a new GetCodeSize object into a buffer, reads from it,
// and checks equality.
func TestGetCodeSizeReadWrite(t *testing.T) {
	_, op1, _ := initGetCodeSize(t)

	op1Buffer := bytes.NewBufferString("")
	err := op1.Write(op1Buffer)
	if err != nil {
		t.Fatalf("error operation write %v", err)
	}

	// read object from buffer
	op2Buffer := bytes.NewBufferString(op1Buffer.String())
	op2, err := ReadGetCodeSize(op2Buffer)
	if err != nil {
		t.Fatalf("failed to read operation. Error: %v", err)
	}
	if op2 == nil {
		t.Fatalf("failed to create newly read operation from buffer")
	}
	// check equivalence
	if !reflect.DeepEqual(op1, op2) {
		t.Fatalf("operations are not the same")
	}
}

// TestGetCodeSizeDebug creates a new GetCodeSize object and checks its Debug message.
func TestGetCodeSizeDebug(t *testing.T) {
	dict, op, addr := initGetCodeSize(t)

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
	label, f := operationLabels[GetCodeSizeID]
	if !f {
		t.Fatalf("label for %d not found", GetCodeSizeID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s\n", label, addr) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestGetCodeSizeExecute creates a new GetCodeSize object and checks its execution signature.
func TestGetCodeSizeExecute(t *testing.T) {
	dict, op, addr := initGetCodeSize(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetCodeSizeID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

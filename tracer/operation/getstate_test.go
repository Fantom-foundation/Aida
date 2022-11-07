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

func initGetState(t *testing.T) (*dict.DictionaryContext, *GetState, common.Address, common.Hash) {
	addr := getRandomAddress(t)
	storage := getRandomAddress(t).Hash()

	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)
	sIdx, _ := dict.EncodeStorage(storage)

	// create new operation
	op := NewGetState(cIdx, sIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetStateID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr, storage
}

// TestGetStateReadWrite writes a new GetState object into a buffer, reads from it,
// and checks equality.
func TestGetStateReadWrite(t *testing.T) {
	_, op1, _, _ := initGetState(t)

	op1Buffer := bytes.NewBufferString("")
	err := op1.Write(op1Buffer)
	if err != nil {
		t.Fatalf("error operation write %v", err)
	}

	// read object from buffer
	op2Buffer := bytes.NewBufferString(op1Buffer.String())
	op2, err := ReadGetState(op2Buffer)
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

// TestGetStateDebug creates a new GetState object and checks its Debug message.
func TestGetStateDebug(t *testing.T) {
	dict, op, addr, storage := initGetState(t)

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
	label, f := operationLabels[GetStateID]
	if !f {
		t.Fatalf("label for %d not found", GetStateID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s, %s\n", label, addr, storage) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestGetStateExecute
func TestGetStateExecute(t *testing.T) {
	dict, op, addr, storage := initGetState(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetStateID, []any{addr, storage}}}
	mock.compareRecordings(expected, t)
}

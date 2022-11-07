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

func initSetState(t *testing.T) (*dict.DictionaryContext, *SetState, common.Address, common.Hash, common.Hash) {
	addr := getRandomAddress(t)
	storage := getRandomAddress(t).Hash()
	value := getRandomAddress(t).Hash()

	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)
	sIdx, _ := dict.EncodeStorage(storage)
	vIdx := dict.EncodeValue(value)

	// create new operation
	op := NewSetState(cIdx, sIdx, vIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SetStateID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr, storage, value
}

// TestSetStateReadWrite writes a new SetState object into a buffer, reads from it,
// and checks equality.
func TestSetStateReadWrite(t *testing.T) {
	_, op1, _, _, _ := initSetState(t)

	op1Buffer := bytes.NewBufferString("")
	err := op1.Write(op1Buffer)
	if err != nil {
		t.Fatalf("error operation write %v", err)
	}

	// read object from buffer
	op2Buffer := bytes.NewBufferString(op1Buffer.String())
	op2, err := ReadSetState(op2Buffer)
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

// TestSetStateDebug creates a new SetState object and checks its Debug message.
func TestSetStateDebug(t *testing.T) {
	dict, op, addr, storage, value := initSetState(t)

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
	label, f := operationLabels[SetStateID]
	if !f {
		t.Fatalf("label for %d not found", SetStateID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s, %s, %s\n", label, addr, storage, value) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestSetStateExecute
func TestSetStateExecute(t *testing.T) {
	dict, op, addr, storage, value := initSetState(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{SetStateID, []any{addr, storage, value}}}
	mock.compareRecordings(expected, t)
}

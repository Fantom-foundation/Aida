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

func initSetStateLcls(t *testing.T) (*dict.DictionaryContext, *SetStateLcls, common.Address, common.Hash, common.Hash) {
	value := getRandomAddress(t).Hash()

	// create dictionary context
	dict := dict.NewDictionaryContext()
	vIdx := dict.EncodeValue(value)

	// create new operation
	op := NewSetStateLcls(vIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SetStateLclsID {
		t.Fatalf("wrong ID returned")
	}

	addr := getRandomAddress(t)
	dict.EncodeContract(addr)

	storage := getRandomAddress(t).Hash()
	dict.EncodeStorage(storage)

	return dict, op, addr, storage, value
}

// TestSetStateLclsReadWrite writes a new SetStateLcls object into a buffer, reads from it,
// and checks equality.
func TestSetStateLclsReadWrite(t *testing.T) {
	_, op1, _, _, _ := initSetStateLcls(t)

	op1Buffer := bytes.NewBufferString("")
	err := op1.Write(op1Buffer)
	if err != nil {
		t.Fatalf("error operation write %v", err)
	}

	// read object from buffer
	op2Buffer := bytes.NewBufferString(op1Buffer.String())
	op2, err := ReadSetStateLcls(op2Buffer)
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

// TestSetStateLclsDebug creates a new SetStateLcls object and checks its Debug message.
func TestSetStateLclsDebug(t *testing.T) {
	dict, op, addr, storage, value := initSetStateLcls(t)

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
	label, f := operationLabels[SetStateLclsID]
	if !f {
		t.Fatalf("label for %d not found", SetStateLclsID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s, %s, %s\n", label, addr, storage, value) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestSetStateLclsExecute
func TestSetStateLclsExecute(t *testing.T) {
	dict, op, addr, storage, value := initSetStateLcls(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{SetStateID, []any{addr, storage, value}}}
	mock.compareRecordings(expected, t)
}

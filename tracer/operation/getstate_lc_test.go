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

func initGetStateLc(t *testing.T) (*dict.DictionaryContext, *GetStateLc, common.Address, common.Hash) {
	storage := getRandomAddress(t).Hash()

	// create dictionary context
	dict := dict.NewDictionaryContext()
	sIdx, _ := dict.EncodeStorage(storage)

	// create new operation
	op := NewGetStateLc(sIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetStateLcID {
		t.Fatalf("wrong ID returned")
	}

	addr := getRandomAddress(t)
	dict.EncodeContract(addr)

	return dict, op, addr, storage
}

// TestGetStateLcReadWrite writes a new GetStateLc object into a buffer, reads from it,
// and checks equality.
func TestGetStateLcReadWrite(t *testing.T) {
	_, op1, _, _ := initGetStateLc(t)

	op1Buffer := bytes.NewBufferString("")
	err := op1.Write(op1Buffer)
	if err != nil {
		t.Fatalf("error operation write %v", err)
	}

	// read object from buffer
	op2Buffer := bytes.NewBufferString(op1Buffer.String())
	op2, err := ReadGetStateLc(op2Buffer)
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

// TestGetStateLcDebug creates a new GetStateLc object and checks its Debug message.
func TestGetStateLcDebug(t *testing.T) {
	dict, op, addr, storage := initGetStateLc(t)

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
	label, f := operationLabels[GetStateLcID]
	if !f {
		t.Fatalf("label for %d not found", GetStateLcID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s, %s\n", label, addr, storage) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestGetStateLcExecute
func TestGetStateLcExecute(t *testing.T) {
	dict, op, addr, storage := initGetStateLc(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetStateID, []any{addr, storage}}}
	mock.compareRecordings(expected, t)
}

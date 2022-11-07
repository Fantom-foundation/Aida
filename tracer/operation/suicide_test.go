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

func initSuicide(t *testing.T) (*dict.DictionaryContext, *Suicide, common.Address) {
	addr := getRandomAddress(t)
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)

	// create new operation
	op := NewSuicide(cIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SuicideID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr
}

// TestSuicideReadWrite writes a new Suicide object into a buffer, reads from it,
// and checks equality.
func TestSuicideReadWrite(t *testing.T) {
	_, op1, _ := initSuicide(t)

	op1Buffer := bytes.NewBufferString("")
	err := op1.Write(op1Buffer)
	if err != nil {
		t.Fatalf("error operation write %v", err)
	}

	// read object from buffer
	op2Buffer := bytes.NewBufferString(op1Buffer.String())
	op2, err := ReadSuicide(op2Buffer)
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

// TestSuicideDebug creates a new Suicide object and checks its Debug message.
func TestSuicideDebug(t *testing.T) {
	dict, op, addr := initSuicide(t)

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
	label, f := operationLabels[SuicideID]
	if !f {
		t.Fatalf("label for %d not found", SuicideID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s\n", label, addr) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestSuicideExecute
func TestSuicideExecute(t *testing.T) {
	dict, op, addr := initSuicide(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{SuicideID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

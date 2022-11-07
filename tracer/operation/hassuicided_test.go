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

func initHasSuicided(t *testing.T) (*dict.DictionaryContext, *HasSuicided, common.Address) {
	addr := getRandomAddress(t)
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)

	// create new operation
	op := NewHasSuicided(cIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != HasSuicidedID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr
}

// TestHasSuicidedReadWrite writes a new HasSuicided object into a buffer, reads from it,
// and checks equality.
func TestHasSuicidedReadWrite(t *testing.T) {
	_, op1, _ := initHasSuicided(t)

	op1Buffer := bytes.NewBufferString("")
	err := op1.Write(op1Buffer)
	if err != nil {
		t.Fatalf("error operation write %v", err)
	}

	// read object from buffer
	op2Buffer := bytes.NewBufferString(op1Buffer.String())
	op2, err := ReadHasSuicided(op2Buffer)
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

// TestHasSuicidedDebug creates a new HasSuicided object and checks its Debug message.
func TestHasSuicidedDebug(t *testing.T) {
	dict, op, addr := initHasSuicided(t)

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
	label, f := operationLabels[HasSuicidedID]
	if !f {
		t.Fatalf("label for %d not found", HasSuicidedID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s\n", label, addr) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestHasSuicidedExecute
func TestHasSuicidedExecute(t *testing.T) {
	dict, op, addr := initHasSuicided(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{HasSuicidedID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

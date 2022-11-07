package operation

import (
	"bytes"
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/ethereum/go-ethereum/common"
	"io"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"
)

func initGetStateLccs(t *testing.T) (*dict.DictionaryContext, *GetStateLccs, common.Address, common.Hash, common.Hash) {
	rand.Seed(time.Now().UnixNano())
	pos := 0

	// create dictionary context
	dict := dict.NewDictionaryContext()

	// create new operation
	op := NewGetStateLccs(pos)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetStateLccsID {
		t.Fatalf("wrong ID returned")
	}

	addr := getRandomAddress(t)
	dict.EncodeContract(addr)

	storage := getRandomAddress(t).Hash()
	dict.EncodeStorage(storage)

	storage2 := getRandomAddress(t).Hash()

	return dict, op, addr, storage, storage2
}

// TestGetStateLccsReadWrite writes a new GetStateLccs object into a buffer, reads from it,
// and checks equality.
func TestGetStateLccsReadWrite(t *testing.T) {
	_, op1, _, _, _ := initGetStateLccs(t)

	op1Buffer := bytes.NewBufferString("")
	err := op1.Write(op1Buffer)
	if err != nil {
		t.Fatalf("error operation write %v", err)
	}

	// read object from buffer
	op2Buffer := bytes.NewBufferString(op1Buffer.String())
	op2, err := ReadGetStateLccs(op2Buffer)
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

// TestGetStateLccsDebug creates a new GetStateLccs object and checks its Debug message.
func TestGetStateLccsDebug(t *testing.T) {
	dict, op, addr, storage, _ := initGetStateLccs(t)

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
	label, f := operationLabels[GetStateLccsID]
	if !f {
		t.Fatalf("label for %d not found", GetStateLccsID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s, %s\n", label, addr, storage) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestGetStateLccsExecute
func TestGetStateLccsExecute(t *testing.T) {
	dict, op, addr, storage, storage2 := initGetStateLccs(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	dict.EncodeStorage(storage2)

	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetStateID, []any{addr, storage}}, {GetStateID, []any{addr, storage2}}}
	mock.compareRecordings(expected, t)
}

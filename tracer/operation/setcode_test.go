package operation

import (
	"bytes"
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/ethereum/go-ethereum/common"
	"io"
	"math/rand"
	"os"
	"testing"
	"time"
)

func initSetCode(t *testing.T) (*dict.DictionaryContext, *SetCode, common.Address, []byte) {
	rand.Seed(time.Now().UnixNano())
	addr := getRandomAddress(t)
	code := make([]byte, 100)
	rand.Read(code)

	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)
	bcIdx := dict.EncodeCode(code)

	// create new operation
	op := NewSetCode(cIdx, bcIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SetCodeID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr, code
}

// TestSetCodeReadWrite writes a new SetCode object into a buffer, reads from it,
// and checks equality.
func TestSetCodeReadWrite(t *testing.T) {
	_, op1, _, _ := initSetCode(t)
	testOperationReadWrite(t, op1, ReadSetCode)
}

// TestSetCodeDebug creates a new SetCode object and checks its Debug message.
func TestSetCodeDebug(t *testing.T) {
	dict, op, addr, value := initSetCode(t)

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
	label, f := operationLabels[SetCodeID]
	if !f {
		t.Fatalf("label for %d not found", SetCodeID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s, %s\n", label, addr, value) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestSetCodeExecute
func TestSetCodeExecute(t *testing.T) {
	dict, op, addr, code := initSetCode(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{SetCodeID, []any{addr, code}}}
	mock.compareRecordings(expected, t)
}

package operation

import (
	"bytes"
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"io"
	"math/rand"
	"os"
	"testing"
	"time"
)

func initBeginBlock(t *testing.T) (*dict.DictionaryContext, *BeginBlock, uint64) {
	rand.Seed(time.Now().UnixNano())
	blId := rand.Uint64()

	// create dictionary context
	dict := dict.NewDictionaryContext()

	// create new operation
	op := NewBeginBlock(blId)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != BeginBlockID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, blId
}

// TestBeginBlockReadWrite writes a new BeginBlock object into a buffer, reads from it,
// and checks equality.
func TestBeginBlockReadWrite(t *testing.T) {
	_, op1, _ := initBeginBlock(t)
	testOperationReadWrite(t, op1, ReadBeginBlock)
}

// TestBeginBlockDebug creates a new BeginBlock object and checks its Debug message.
func TestBeginBlockDebug(t *testing.T) {
	dict, op, value := initBeginBlock(t)

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
	label, f := operationLabels[BeginBlockID]
	if !f {
		t.Fatalf("label for %d not found", BeginBlockID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %d\n", label, value) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestBeginBlockExecute
func TestBeginBlockExecute(t *testing.T) {
	dict, op, _ := initBeginBlock(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	mock.compareRecordings([]Record{}, t)
	// currently BeginBlock isn't recorded
	//expected := []Record{{BeginBlockID, []any{blId}}}
	//mock.compareRecordings(expected, t)
}

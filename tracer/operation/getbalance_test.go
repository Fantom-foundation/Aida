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

func initGetBalance(t *testing.T) (*dict.DictionaryContext, *GetBalance, common.Address) {
	addr := getRandomAddress(t)
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)

	// create new operation
	op := NewGetBalance(cIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetBalanceID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr
}

// TestGetBalanceReadWrite writes a new GetBalance object into a buffer, reads from it,
// and checks equality.
func TestGetBalanceReadWrite(t *testing.T) {
	_, op1, _ := initGetBalance(t)

	op1Buffer := bytes.NewBufferString("")
	err := op1.Write(op1Buffer)
	if err != nil {
		t.Fatalf("error operation write %v", err)
	}

	// read object from buffer
	op2Buffer := bytes.NewBufferString(op1Buffer.String())
	op2, err := ReadGetBalance(op2Buffer)
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

// TestGetBalanceDebug creates a new GetBalance object and checks its Debug message.
func TestGetBalanceDebug(t *testing.T) {
	dict, op, addr := initGetBalance(t)

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
	label, f := operationLabels[GetBalanceID]
	if !f {
		t.Fatalf("label for %d not found", GetBalanceID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s\n", label, addr) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestGetBalanceExecute
func TestGetBalanceExecute(t *testing.T) {
	dict, op, addr := initGetBalance(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetBalanceID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

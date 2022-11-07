package operation

import (
	"bytes"
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/ethereum/go-ethereum/common"
	"io"
	"os"
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
	testOperationReadWrite(t, op1, ReadGetBalance)
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

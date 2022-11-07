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

func initCreateAccount(t *testing.T) (*dict.DictionaryContext, *CreateAccount, common.Address) {
	addr := getRandomAddress(t)
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)

	// create new operation
	op := NewCreateAccount(cIdx)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != CreateAccountID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr
}

// TestCreateAccountReadWrite writes a new CreateAccount object into a buffer, reads from it,
// and checks equality.
func TestCreateAccountReadWrite(t *testing.T) {
	_, op1, _ := initCreateAccount(t)
	testOperationReadWrite(t, op1, ReadCreateAccount)
}

// TestCreateAccountDebug creates a new CreateAccount object and checks its Debug message.
func TestCreateAccountDebug(t *testing.T) {
	dict, op, addr := initCreateAccount(t)

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
	label, f := operationLabels[CreateAccountID]
	if !f {
		t.Fatalf("label for %d not found", CreateAccountID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s\n", label, addr) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestCreateAccountExecute
func TestCreateAccountExecute(t *testing.T) {
	dict, op, addr := initCreateAccount(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{CreateAccountID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

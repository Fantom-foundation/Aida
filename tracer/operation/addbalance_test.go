package operation

import (
	"bytes"
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/ethereum/go-ethereum/common"
	"io"
	"math/big"
	"math/rand"
	"os"
	"testing"
	"time"
)

func initAddBalance(t *testing.T) (*dict.DictionaryContext, *AddBalance, common.Address, *big.Int) {
	rand.Seed(time.Now().UnixNano())
	addr := getRandomAddress(t)
	value := big.NewInt(rand.Int63n(100000))
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)

	// create new operation
	op := NewAddBalance(cIdx, value)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != AddBalanceID {
		t.Fatalf("wrong ID returned")
	}
	return dict, op, addr, value
}

// TestAddBalanceReadWrite writes a new AddBalance object into a buffer, reads from it,
// and checks equality.
func TestAddBalanceReadWrite(t *testing.T) {
	_, op1, _, _ := initAddBalance(t)
	testOperationReadWrite(t, op1, ReadAddBalance)
}

// TestAddBalanceDebug creates a new AddBalance object and checks its Debug message.
func TestAddBalanceDebug(t *testing.T) {
	dict, op, addr, value := initAddBalance(t)

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
	label, f := operationLabels[AddBalanceID]
	if !f {
		t.Fatalf("label for %d not found", AddBalanceID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s, %s\n", label, addr, value) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestAddBalanceExecute
func TestAddBalanceExecute(t *testing.T) {
	dict, op, addr, value := initAddBalance(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{AddBalanceID, []any{addr, value}}}
	mock.compareRecordings(expected, t)
}

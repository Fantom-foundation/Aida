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
	"reflect"
	"testing"
	"time"
)

func initSubBalance(t *testing.T) (*dict.DictionaryContext, *SubBalance, common.Address, *big.Int) {
	rand.Seed(time.Now().UnixNano())
	addr := getRandomAddress(t)
	value := big.NewInt(rand.Int63n(100000))
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(addr)

	// create new operation
	op := NewSubBalance(cIdx, value)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SubBalanceID {
		t.Fatalf("wrong ID returned")
	}
	return dict, op, addr, value
}

// TestSubBalanceReadWrite writes a new SubBalance object into a buffer, reads from it,
// and checks equality.
func TestSubBalanceReadWrite(t *testing.T) {
	_, op1, _, _ := initSubBalance(t)

	op1Buffer := bytes.NewBufferString("")
	err := op1.Write(op1Buffer)
	if err != nil {
		t.Fatalf("error operation write %v", err)
	}

	// read object from buffer
	op2Buffer := bytes.NewBufferString(op1Buffer.String())
	op2, err := ReadSubBalance(op2Buffer)
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

// TestSubBalanceDebug creates a new SubBalance object and checks its Debug message.
func TestSubBalanceDebug(t *testing.T) {
	dict, op, addr, value := initSubBalance(t)

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
	label, f := operationLabels[SubBalanceID]
	if !f {
		t.Fatalf("label for %d not found", SubBalanceID)
	}

	if buf.String() != fmt.Sprintf("\t%s: %s, %s\n", label, addr, value) {
		t.Fatalf("wrong debug message: %s", buf.String())
	}
}

// TestSubBalanceExecute
func TestSubBalanceExecute(t *testing.T) {
	dict, op, addr, value := initSubBalance(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{SubBalanceID, []any{addr, value}}}
	mock.compareRecordings(expected, t)
}

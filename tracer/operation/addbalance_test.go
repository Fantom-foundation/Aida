package operation

import (
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"math/rand"
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
	testOperationDebug(t, dict, op, AddBalanceID, func(label string) string {
		return fmt.Sprintf("\t%s: %s, %s\n", label, addr, value)
	})
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

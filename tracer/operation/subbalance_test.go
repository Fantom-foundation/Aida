package operation

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initSubBalance(t *testing.T) (*context.Replay, *SubBalance, common.Address, *big.Int) {
	rand.Seed(time.Now().UnixNano())
	addr := getRandomAddress(t)
	value := big.NewInt(rand.Int63n(100000))
	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)

	// create new operation
	op := NewSubBalance(contract, value)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != SubBalanceID {
		t.Fatalf("wrong ID returned")
	}
	return ctx, op, addr, value
}

// TestSubBalanceReadWrite writes a new SubBalance object into a buffer, reads from it,
// and checks equality.
func TestSubBalanceReadWrite(t *testing.T) {
	_, op1, _, _ := initSubBalance(t)
	testOperationReadWrite(t, op1, ReadSubBalance)
}

// TestSubBalanceDebug creates a new SubBalance object and checks its Debug message.
func TestSubBalanceDebug(t *testing.T) {
	ctx, op, addr, value := initSubBalance(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr, value))
}

// TestSubBalanceExecute
func TestSubBalanceExecute(t *testing.T) {
	ctx, op, addr, value := initSubBalance(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{SubBalanceID, []any{addr, value}}}
	mock.compareRecordings(expected, t)
}

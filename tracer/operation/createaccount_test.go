package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initCreateAccount(t *testing.T) (*context.Replay, *CreateAccount, common.Address) {
	addr := getRandomAddress(t)
	// create context context
	ctx := context.NewReplay()
	contract := ctx.EncodeContract(addr)

	// create new operation
	op := NewCreateAccount(contract)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != CreateAccountID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, addr
}

// TestCreateAccountReadWrite writes a new CreateAccount object into a buffer, reads from it,
// and checks equality.
func TestCreateAccountReadWrite(t *testing.T) {
	_, op1, _ := initCreateAccount(t)
	testOperationReadWrite(t, op1, ReadCreateAccount)
}

// TestCreateAccountDebug creates a new CreateAccount object and checks its Debug message.
func TestCreateAccountDebug(t *testing.T) {
	ctx, op, addr := initCreateAccount(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr))

}

// TestCreateAccountExecute
func TestCreateAccountExecute(t *testing.T) {
	ctx, op, addr := initCreateAccount(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{CreateAccountID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

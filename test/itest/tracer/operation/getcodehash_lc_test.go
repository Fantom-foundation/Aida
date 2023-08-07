package operation

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/rc-testing/test/itest/tracer/context"
	"github.com/ethereum/go-ethereum/common"
)

func initGetCodeHashLc(t *testing.T) (*context.Replay, *GetCodeHashLc, common.Address) {
	// create context context
	ctx := context.NewReplay()

	addr := getRandomAddress(t)
	ctx.EncodeContract(addr)

	// create new operation
	op := NewGetCodeHashLc()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCodeHashLcID {
		t.Fatalf("wrong ID returned")
	}

	return ctx, op, addr
}

// TestGetCodeHashLcReadWrite writes a new GetCodeHashLc object into a buffer, reads from it,
// and checks equality.
func TestGetCodeHashLcReadWrite(t *testing.T) {
	_, op1, _ := initGetCodeHashLc(t)
	testOperationReadWrite(t, op1, ReadGetCodeHashLc)
}

// TestGetCodeHashLcDebug creates a new GetCodeHashLc object and checks its Debug message.
func TestGetCodeHashLcDebug(t *testing.T) {
	ctx, op, addr := initGetCodeHashLc(t)
	testOperationDebug(t, ctx, op, fmt.Sprint(addr))
}

// TestGetCodeHashLcExecute
func TestGetCodeHashLcExecute(t *testing.T) {
	ctx, op, addr := initGetCodeHashLc(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, ctx)

	// check whether methods were correctly called
	expected := []Record{{GetCodeHashID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

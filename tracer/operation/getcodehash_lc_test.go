package operation

import (
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func initGetCodeHashLc(t *testing.T) (*dict.DictionaryContext, *GetCodeHashLc, common.Address) {
	// create dictionary context
	dict := dict.NewDictionaryContext()

	addr := getRandomAddress(t)
	dict.EncodeContract(addr)

	// create new operation
	op := NewGetCodeHashLc()
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != GetCodeHashLcID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, addr
}

// TestGetCodeHashLcReadWrite writes a new GetCodeHashLc object into a buffer, reads from it,
// and checks equality.
func TestGetCodeHashLcReadWrite(t *testing.T) {
	_, op1, _ := initGetCodeHashLc(t)
	testOperationReadWrite(t, op1, ReadGetCodeHashLc)
}

// TestGetCodeHashLcDebug creates a new GetCodeHashLc object and checks its Debug message.
func TestGetCodeHashLcDebug(t *testing.T) {
	dict, op, addr := initGetCodeHashLc(t)
	testOperationDebug(t, dict, op, GetCodeHashLcID, func(label string) string {
		return fmt.Sprintf("\t%s: %s\n", label, addr)
	})
}

// TestGetCodeHashLcExecute
func TestGetCodeHashLcExecute(t *testing.T) {
	dict, op, addr := initGetCodeHashLc(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{GetCodeHashID, []any{addr}}}
	mock.compareRecordings(expected, t)
}

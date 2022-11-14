package operation

import (
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"math/rand"
	"testing"
	"time"
)

func initCommit(t *testing.T) (*dict.DictionaryContext, *Commit, bool) {
	rand.Seed(time.Now().UnixNano())
	deleteEmpty := rand.Intn(2) == 1
	// create dictionary context
	dict := dict.NewDictionaryContext()

	// create new operation
	op := NewCommit(deleteEmpty)
	if op == nil {
		t.Fatalf("failed to create operation")
	}
	// check id
	if op.GetId() != CommitID {
		t.Fatalf("wrong ID returned")
	}

	return dict, op, deleteEmpty
}

// TestCommitReadWrite writes a new Commit object into a buffer, reads from it,
// and checks equality.
func TestCommitReadWrite(t *testing.T) {
	_, op1, _ := initCommit(t)
	testOperationReadWrite(t, op1, ReadCommit)
}

// TestCommitDebug creates a new Commit object and checks its Debug message.
func TestCommitDebug(t *testing.T) {
	dict, op, deleteEmpty := initCommit(t)
	testOperationDebug(t, dict, op, fmt.Sprint(deleteEmpty))
}

// TestCommitExecute
func TestCommitExecute(t *testing.T) {
	dict, op, deleteEmpty := initCommit(t)

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)

	// check whether methods were correctly called
	expected := []Record{{CommitID, []any{deleteEmpty}}}
	mock.compareRecordings(expected, t)
}

package snapshot

import (
	"context"
	"testing"

	"github.com/Fantom-foundation/rc-testing/test/vmtest/world-state/types"
)

func TestStateDB_CloneTo(t *testing.T) {
	dl, ok := t.Deadline()
	if !ok {
		t.Fatalf("test deadline exceeded")
	}

	// compare the DB to itself
	ctx, cancel := context.WithDeadline(context.Background(), dl)
	defer cancel()

	// prep source DB
	fromDB, nodes, _, _ := makeTestDB(t)

	// create target in-memory database
	toDB, err := OpenStateDB("")
	if err != nil {
		t.Fatalf("failed test data build; could not create empty target DB; %s", err.Error())
	}

	var count int
	err = fromDB.Copy(ctx, toDB, func(a *types.Account) {
		count++

		// make sure the account is one of the known
		b, ok := nodes[a.Hash]
		if !ok {
			t.Fatalf("failed clone test; expected known account, got unknown %s", a.Hash.String())
			return
		}

		if !a.IsIdentical(&b) {
			err = a.IsDifferent(&b)
			t.Fatalf("failed clone test; expected identical account, got %s", err.Error())
		}
	})
	if err != nil {
		t.Fatalf("failed clone test; expected no error, got %s", err.Error())
	}

	if count != len(nodes) {
		t.Fatalf("failed clone test; expected %d accounts, got %d", len(nodes), count)
	}

	MustCloseStateDB(fromDB)
	MustCloseStateDB(toDB)
}

// TestStateDB_CloneTo_CtxFail tests Copy function in the context expiration state
func TestStateDB_CloneTo_CtxFail(t *testing.T) {
	// context expiration
	ctx, cancel := context.WithTimeout(context.Background(), cCtxNoTime)
	defer cancel()

	// prep source DB
	fromDB, _, _, _ := makeTestDB(t)

	// create target in-memory database
	toDB, err := OpenStateDB("")
	if err != nil {
		t.Fatalf("failed test data build; could not create empty target DB; %s", err.Error())
	}

	err = fromDB.Copy(ctx, toDB, nil)
	if err != context.DeadlineExceeded {
		t.Errorf("failed clone test on context expiration; expected DeadlineExceeded error, got %s", errorStr(err))
	}

	MustCloseStateDB(fromDB)
	MustCloseStateDB(toDB)
}

func TestStateDB_CloneToAndCompare(t *testing.T) {
	ctx := context.Background()

	// prep source DB
	fromDB, _, _, _ := makeTestDB(t)

	// create target in-memory database
	toDB, err := OpenStateDB("")
	if err != nil {
		t.Fatalf("failed test data build; could not create empty target DB; %s", err.Error())
	}

	err = fromDB.Copy(ctx, toDB, nil)
	if err != nil {
		t.Fatalf("failed clone test; expected no error, got %s", err.Error())
	}

	err = fromDB.CompareTo(context.Background(), toDB)
	if err != nil {
		t.Fatalf("failed DB cloning; the target DB is not identical to the original DB; %s", err.Error())
	}

	err = toDB.CompareTo(context.Background(), fromDB)
	if err != nil {
		t.Fatalf("failed DB cloning; the original DB is not identical to the target DB; %s", err.Error())
	}

	MustCloseStateDB(fromDB)
	MustCloseStateDB(toDB)
}

package snapshot

import (
	"context"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"testing"
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
	fromDB, nodes, _ := makeTestDB(t)

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

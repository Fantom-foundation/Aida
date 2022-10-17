// Package snapshot implements database interfaces for the world state manager.
package snapshot

import (
	"context"
	"testing"
)

func TestStateDB_CompareTo(t *testing.T) {
	dl, ok := t.Deadline()
	if !ok {
		t.Fatalf("test deadline exceeded")
	}

	// compare the DB to itself
	ctx, cancel := context.WithDeadline(context.Background(), dl)
	defer cancel()

	dba, _, _ := makeTestDB(t)
	err := dba.CompareTo(ctx, dba)
	if err != nil {
		t.Errorf("failed identical DB comparison; expected no error, got %s", err.Error())
	}

	// compare the DB to another one
	dbb, _, _ := makeTestDB(t)
	err = dba.CompareTo(ctx, dbb)
	if err == nil {
		t.Errorf("failed different DB comparison; expected to receive error, got none")
	}
}

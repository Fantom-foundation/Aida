// Package snapshot implements database interfaces for the world state manager.
package snapshot

import (
	"context"
	"testing"
	"time"
)

// context timeout options
const (
	cCtxEnoughTime = 30 * time.Second     // enough time to perform a function
	cCtxNoTime     = 0 * time.Millisecond // little time for function execution
)

// compareTo_SameDB executes CompareTo function by comparing db with itself
func compareTo_SameDB(d time.Duration, t *testing.T) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	// compare the DB to itself
	dba, _, _, _ := makeTestDB(t)
	err := dba.CompareTo(ctx, dba)
	MustCloseStateDB(dba)

	return err
}

// compareTo_DifferentDB executes CompareTo function by comparing db with another db
func compareTo_DifferentDB(d time.Duration, t *testing.T) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	// compare the DB to another one
	dba, _, _, _ := makeTestDB(t)
	dbb, _, _, _ := makeTestDB(t)
	err := dba.CompareTo(ctx, dbb)

	MustCloseStateDB(dba)
	MustCloseStateDB(dbb)

	return err
}

// TestStateDB_CompareTo_SameDB tests CompareTo function by comparing db with itself
func TestStateDB_CompareTo_SameDB(t *testing.T) {
	err := compareTo_SameDB(cCtxEnoughTime, t)
	if err != nil {
		t.Errorf("failed identical DB comparison; expected no error, got %s", err.Error())
	}
}

// TestStateDB_CompareTo_SameDB_CtxFail tests CompareTo function in the context expiration state
func TestStateDB_CompareTo_SameDB_CtxFail(t *testing.T) {
	err := compareTo_SameDB(cCtxNoTime, t)
	if err != context.DeadlineExceeded {
		t.Errorf("failed identical DB comparison on context expiration; expected DeadlineExceeded error, got %s", errorStr(err))
	}
}

// TestStateDB_CompareTo_DifferentDB tests CompareTo function by comparing db with another db
func TestStateDB_CompareTo_DifferentDB(t *testing.T) {
	err := compareTo_DifferentDB(cCtxEnoughTime, t)
	if err == nil {
		t.Error("failed different DB comparison; expected to receive error, got none")
	}
}

// TestStateDB_CompareTo_DifferentDB_CtxFail tests CompareTo function in the context expiration state
func TestStateDB_CompareTo_DifferentDB_CtxFail(t *testing.T) {
	err := compareTo_DifferentDB(0, t)
	if err != context.DeadlineExceeded {
		t.Errorf("failed different DB comparison on context expiration; expected DeadlineExceeded error, got %s", errorStr(err))
	}
}

// errorStr returns a string with the description of the error
func errorStr(err error) string {
	if err == nil {
		return "none"
	} else {
		return err.Error()
	}
}

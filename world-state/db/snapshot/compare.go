// Package snapshot implements database interfaces for the world state manager.
package snapshot

import (
	"context"
	"fmt"
	"sync"
)

// CompareTo compares the world state snapshot database
// with the given target DB returning NIL if the databases are identical,
// and error of the first found difference, if there is any.
func (db *StateDB) CompareTo(ctx context.Context, target *StateDB) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// collect errors from both
	fail := make(chan error, 2)
	defer close(fail)

	wg := new(sync.WaitGroup)
	wg.Add(2)
	go checkAccounts(ctx, cancel, wg, db, target, fail)
	go checkAccounts(ctx, cancel, wg, target, db, fail)

	// wait for the check to finish
	wg.Wait()

	// any error received?
	select {
	case err := <-fail:
		return err
	default:
	}

	return nil
}

// checkAccounts runs accounts check between source and destination databases.
func checkAccounts(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, src, dst *StateDB, fail chan error) {
	defer wg.Done()

	// compare and register error in the fail channel; context will be canceled if an error happens
	err := src.compareAllAccounts(ctx, dst)
	if err != nil {
		fail <- err
		cancel()
	}
}

// checkAllAccounts iterates all the accounts in our database and
func (db *StateDB) compareAllAccounts(ctx context.Context, target *StateDB) error {
	iter := db.NewAccountIterator(ctx)
	defer iter.Release()

	// loop over all the accounts
	for iter.Next() {
		// make sure to check the context status
		select {
		case <-ctx.Done():
			break
		default:
		}

		// get the other account
		other, err := target.AccountByHash(iter.Value().Hash)
		if err != nil {
			return fmt.Errorf("target account not loaded; %s", err.Error())
		}

		// compare accounts
		err = other.IsDifferent(iter.Value())
		if err != nil {
			return fmt.Errorf("target account different; %s", err.Error())
		}
	}

	return nil
}

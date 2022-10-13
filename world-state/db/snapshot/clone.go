// Package snapshot implements database interfaces for the world state manager.
package snapshot

import (
	"context"
	"fmt"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
)

// Copy creates a copy of the state snapshot database to the given output handle.
// The copy does not erase previous data from the target database.
// If you want a clean copy, make sure you use an empty DB.
func (db *StateDB) Copy(ctx context.Context, to *StateDB, onAccount func(*types.Account)) error {
	// make a buffer for reader/writer account exchange
	wb := make(chan types.Account, 100)
	defer func() {
		close(wb)
	}()

	// store data to the target database
	wFail := NewQueueWriter(ctx, to, wb)

	// we will use iterator to get all the source accounts
	it := db.NewAccountIterator(ctx)
	defer it.Release()

	// iterate source database
	for it.Next() {
		acc := it.Value()
		if it.Error() != nil {
			break
		}

		select {
		case err := <-wFail:
			if err != nil {
				return err
			}
		case wb <- *acc:
			if onAccount != nil {
				onAccount(acc)
			}
		}
	}

	// release resources
	return it.Error()
}

// NewQueueWriter creates a writer thread, which inserts Accounts from an input queue into the given database.
func NewQueueWriter(ctx context.Context, db *StateDB, in chan types.Account) chan error {
	e := make(chan error, 1)

	go func(fail chan error) {
		defer close(fail)
		for {
			// get all the found accounts from the input channel
			select {
			case <-ctx.Done():
				return
			case account, open := <-in:
				if !open {
					break
				}

				// insert account data
				err := db.PutAccount(&account)
				if err != nil {
					fail <- fmt.Errorf("can not write account %s; %s\n", account.Hash.String(), err.Error())
					return
				}
			}
		}
	}(e)

	return e
}

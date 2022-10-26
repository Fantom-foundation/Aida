// Package snapshot implements database interfaces for the world state manager.
package snapshot

import (
	"context"
	"github.com/Fantom-foundation/Aida/world-state/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

// AccountIterator implements an iterator over the whole account space in the snapshot database.
type AccountIterator struct {
	ctx     context.Context
	err     error
	iter    ethdb.Iterator
	item    *accountIteratorItem
	items   chan accountIteratorItem
	closed  chan any
	decoder func(key []byte, data []byte) (*types.Account, error)
}

// accountIteratorItem represents an item in account iteration.
type accountIteratorItem struct {
	key []byte
	acc *types.Account
}

// NewAccountIterator creates a new iterator for traversing account space in state snapshot DB.
func (db *StateDB) NewAccountIterator(ctx context.Context) *AccountIterator {
	ai := AccountIterator{
		ctx:     ctx,
		iter:    db.Backend.NewIterator(AccountPrefix, nil),
		items:   make(chan accountIteratorItem, 50),
		closed:  make(chan any),
		decoder: db.decodeAccount,
	}

	go ai.load()
	return &ai
}

// load accounts from the backend into the processing queue until terminated or errored.
func (ai *AccountIterator) load() {
	defer close(ai.items)
	for {
		// do we have another available item?
		if !ai.iter.Next() {
			return
		}

		// decode data
		acc, err := ai.decoder(ai.iter.Key(), ai.iter.Value())
		if err != nil {
			ai.err = err
			return
		}

		// capture any error
		if ai.iter.Error() != nil {
			ai.err = ai.iter.Error()
			return
		}

		// push the item to queue
		itm := accountIteratorItem{
			key: acc.Hash.Bytes(),
			acc: acc,
		}

		select {
		case <-ai.ctx.Done():
			ai.err = ai.ctx.Err()
			return
		case <-ai.closed:
			return
		case ai.items <- itm:
		}
	}
}

// Next moves the iterator to the next key/value pair. It returns whether the
// iterator is exhausted.
func (ai *AccountIterator) Next() bool {
	select {
	case <-ai.ctx.Done():
		ai.err = ai.ctx.Err()
	case <-ai.closed:
	case itm, open := <-ai.items:
		if open {
			ai.item = &itm
			return true
		}
	}
	return false
}

// Release releases associated resources. Release should always succeed and can
// be called multiple times without causing error.
func (ai *AccountIterator) Release() {
	select {
	case <-ai.closed:
	default:
		close(ai.closed)
	}
}

// Key returns the key of the current key/value pair, or nil if done. The caller
// should not modify the contents of the returned slice, and its contents may
// change on the next call to Next.
func (ai *AccountIterator) Key() []byte {
	if ai.item == nil {
		return nil
	}
	return ai.item.key
}

// Value returns the value of the current key/value pair, or nil if done.
// Caller can modify the value of returned account.
func (ai *AccountIterator) Value() *types.Account {
	if ai.item == nil {
		return nil
	}
	return ai.item.acc
}

// Error returns any accumulated error. Exhausting all the key/value pairs
// is not considered to be an error.
func (ai *AccountIterator) Error() error {
	return ai.err
}

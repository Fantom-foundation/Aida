// Package snapshot implements database interfaces for the world state manager.
package snapshot

import (
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

// AccountIterator implements an iterator over the whole account space in the snapshot database.
type AccountIterator struct {
	ethdb.Iterator
	err     error
	decoder func(key []byte, data []byte) (*types.Account, error)
}

// NewAccountIterator creates a new iterator for traversing account space in state snapshot DB.
func (db *StateDB) NewAccountIterator() *AccountIterator {
	return &AccountIterator{
		decoder:  db.decodeAccount,
		Iterator: db.Backend.NewIterator([]byte{AccountPrefix}, nil),
	}
}

// Key returns the key of the current key/value pair, or nil if done. The caller
// should not modify the contents of the returned slice, and its contents may
// change on the next call to Next.
func (ai *AccountIterator) Key() []byte {
	return ai.Iterator.Key()[1:]
}

// Value returns the value of the current key/value pair, or nil if done.
// Caller can modify the value of returned account.
func (ai *AccountIterator) Value() *types.Account {
	acc, err := ai.decoder(ai.Iterator.Key(), ai.Iterator.Value())
	if err != nil {
		ai.err = err
		return nil
	}
	return acc
}

// Error returns any accumulated error. Exhausting all the key/value pairs
// is not considered to be an error.
func (ai *AccountIterator) Error() error {
	if ai.err != nil {
		return ai.err
	}
	return ai.Iterator.Error()
}

// Package dump implements world state trie dump into a snapshot database.
package dump

import (
	"github.com/Fantom-foundation/Aida-Testing/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"log"
)

// dbWriter inserts received Accounts into database
func dbWriter(db *snapshot.StateDB, in chan types.Account) {
	for {
		// get all the found accounts from the input channel
		account, ok := <-in
		if !ok {
			return
		}

		// insert account data
		err := db.PutAccount(&account)
		if err != nil {
			log.Printf("can not write account %s; %s\n", account.Hash.String(), err.Error())
		}
	}
}

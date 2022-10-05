// Package dump implements world state trie dump into a snapshot database.
package dump

import (
	"github.com/Fantom-foundation/Aida-Testing/world-state/db"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"github.com/status-im/keycard-go/hexutils"
	"log"
)

// dbWriter inserts received Accounts into database
func dbWriter(db *db.StateSnapshotDB, in chan types.Account) {
	for {
		// get all the found accounts from the input channel
		account, ok := <-in
		if !ok {
			return
		}

		// insert account code into database in separate record
		err := db.PutCode(account.Code)
		if err != nil {
			log.Printf("can not write code %s; %s\n", hexutils.BytesToHex(account.CodeHash), err.Error())
		}

		// insert account data
		err = db.PutAccount(&account)
		if err != nil {
			log.Printf("can not write account %s; %s\n", account.Hash.String(), err.Error())
		}
	}
}

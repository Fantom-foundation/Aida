// Package dump implements world state trie dump into a snapshot database.
package dump

import (
	"context"
	"github.com/Fantom-foundation/Aida-Testing/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
)

// dbWriter inserts received Accounts into database
func dbWriter(ctx context.Context, db *snapshot.StateDB, in chan types.Account, log Logger) {
	defer log.Infof("DB writer done")

	e := snapshot.NewQueueWriter(ctx, db, in)
	err := <-e
	if err != nil {
		log.Errorf(err.Error())
	}
}

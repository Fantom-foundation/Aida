package snapshot

import (
	"context"

	substate "github.com/Fantom-foundation/Substate"

	"github.com/ethereum/go-ethereum/common"
)

// ToSubstateAlloc converts snapshot world state database into SubstateAlloc format
func (db *StateDB) ToSubstateAlloc(ctx context.Context) (substate.SubstateAlloc, error) {
	ssAccounts := make(substate.SubstateAlloc)
	iter := db.NewAccountIterator(ctx)
	defer iter.Release()

	needed := map[common.Hash]bool{}

	// loop over all the accounts
	for iter.Next() {
		if iter.Error() != nil {
			break
		}

		// make sure to check the context status
		select {
		case <-ctx.Done():
			break
		default:
		}

		acc := iter.Value()
		address, err := db.HashToAccountAddress(acc.Hash)
		if err != nil {
			// not all storage addresses are currently exportable - missing pre genesis data
			//return nil, fmt.Errorf("target storage %s not found; %s", acc.Hash.String(), err.Error())
			continue
		}
		storage := make(map[common.Hash]common.Hash, len(acc.Storage))
		for h, v := range acc.Storage {
			// We use the hashed keys in the first iteration, before resolving them in a bulk fetch
			// from the DB and rewritting them below.
			needed[h] = true
			storage[h] = v
		}

		ss := substate.SubstateAccount{
			Nonce:   acc.Nonce,
			Balance: acc.Balance,
			Storage: storage,
			Code:    acc.Code,
		}

		ssAccounts[address] = &ss
	}

	if iter.Error() != nil {
		return nil, iter.Error()
	}

	// Resolve all hashed slot addresses in one go.
	resolved, err := db.HashesToStorage(needed)
	if err != nil {
		return nil, err
	}

	// Rewrite storage keys according to resolved keys.
	for _, value := range ssAccounts {
		storage := make(map[common.Hash]common.Hash, len(value.Storage))
		for h, v := range value.Storage {
			s, found := resolved[h]
			if found {
				storage[s] = v
			} else {
				// not all storage addresses are currently exportable - missing pre genesis data
				//return nil, fmt.Errorf("target storage %s not found; %s", acc.Hash.String(), err.Error())
			}
		}
		value.Storage = storage
	}

	return ssAccounts, nil
}

package snapshot

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/substate"
)

// SubstateAlloc extract substate database
func (db *StateDB) SubstateAlloc(ctx context.Context) (substate.SubstateAlloc, error) {
	ssAccounts := make(substate.SubstateAlloc)
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

		acc := iter.Value()
		address, err := db.HashToAccountAddress(acc.Hash)
		if err != nil {
			// not all storage addresses are currently exportable - missing pre genesis data
			//return nil, fmt.Errorf("target storage %s not found; %s", acc.Hash.String(), err.Error())
			continue
		}
		storage := make(map[common.Hash]common.Hash, len(acc.Storage))
		for h, v := range acc.Storage {
			s, err := db.HashToStorage(h)
			if err != nil {
				// not all storage addresses are currently exportable - missing pre genesis data
				//return nil, fmt.Errorf("target storage %s not found; %s", acc.Hash.String(), err.Error())
				continue
			}
			storage[s] = v
		}

		ss := substate.SubstateAccount{
			Nonce:   acc.Nonce,
			Balance: acc.Balance,
			Storage: storage,
			Code:    acc.Code,
		}

		ssAccounts[address] = &ss
	}

	return ssAccounts, nil
}

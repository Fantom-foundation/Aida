package snapshot

import (
	"fmt"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/rc-testing/test/vmtest/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// EvolveState iterates trough Substates between first and target blocks
// anticipates that SubstateDB is already open
func EvolveState(stateDB *StateDB, firstBlock uint64, targetBlock uint64, workers int, progress func(uint64), validate func(error)) (uint64, error) {
	// contains last block id
	var lastProcessedBlock uint64 = 0

	// iterator starting from first block - current block of stateDB
	iter := substate.NewSubstateIterator(firstBlock, workers)
	defer iter.Release()

	// iteration trough substates
	for iter.Next() {
		tx := iter.Value()
		if tx.Block > targetBlock {
			break
		}

		// log progress
		if progress != nil {
			progress(tx.Block)
		}

		// EvolveState of database by single Substate Output values
		err := evolveSubstate(tx, stateDB, validate)
		if err != nil {
			return 0, err
		}
		lastProcessedBlock = tx.Block
	}

	return lastProcessedBlock, nil
}

// evolveSubstate evolves world state db supplied substate.substateOut containing data of accounts at the end of one transaction
func evolveSubstate(tx *substate.Transaction, stateDB *StateDB, validate func(error)) error {
	sub := tx.Substate
	// validation of InputAlloc state
	if validate != nil {
		for address, substateAccount := range sub.InputAlloc {
			acc, err := stateDB.Account(address)
			if err != nil {
				validate(fmt.Errorf("%d - %s not found in database", tx.Block, address.String()))
			}
			acc.IsDifferentToSubstate(substateAccount, tx.Block, address.String(), validate)
		}
	}

	for address, substateAccount := range sub.OutputAlloc {
		// get account stored in state snapshot database
		acc, err := stateDB.Account(address)
		if err != nil {
			// account was not found in database therefore we need to create new instance
			addrHash := crypto.Keccak256Hash(address.Bytes())
			acc = &types.Account{Hash: addrHash}

			if len(substateAccount.Storage) > 0 {
				acc.Storage = make(map[common.Hash]common.Hash, len(substateAccount.Storage))
			}
		}

		// updating account data
		acc.Code = substateAccount.Code
		acc.Nonce = substateAccount.Nonce
		acc.Balance = substateAccount.Balance

		// updating account storage
		updateStorage(acc, substateAccount)

		// inserting updated account into database
		err = stateDB.PutAccount(acc)
		if err != nil {
			return fmt.Errorf("unable to insert account %s in database; %s", address.String(), err.Error())
		}

	}
	return nil
}

// updateStorage updates account with substateAccount storage records
func updateStorage(acc *types.Account, substateAccount *substate.SubstateAccount) {
	// overwriting all changed values in storage
	for keyRaw, value := range substateAccount.Storage {
		// generation of key
		// keyRaw consists of unhashed ordered keys
		// eg. keyRaw=0x0000000000000000000000000000000000000000000000000000000000000001 (substate record key)
		// 	   key=0xb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6 (snapshot record key)
		key := common.BytesToHash(crypto.Keccak256(keyRaw.Bytes()))
		if value == ZeroHash {
			if _, found := acc.Storage[key]; found {
				// removing key with empty value from storage
				delete(acc.Storage, key)
			}
			continue
		}
		// storing new value or updating old value
		acc.Storage[key] = value
	}
}

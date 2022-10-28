package trace

import (
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/state"
	"github.com/ethereum/go-ethereum/substate"
)

// validateDatabase validates whether the world-state is contained in the db object
// NB: We can only check what must be in the db (but cannot check whether db stores more)
// Perhaps reuse some of the code from
func validateStateDB(ws substate.SubstateAlloc, db state.StateDB) error {
	for addr, account := range ws {
		if !db.Exist(addr) {
			return fmt.Errorf("Account %v does not exist", addr.Hex())
		}
		if account.Balance.Cmp(db.GetBalance(addr)) != 0 {
			return fmt.Errorf("Failed to validate balance for account %v\n"+
				"\twant %v\n"+
				"\thave %v",
				addr.Hex(), account.Balance, db.GetBalance(addr))
		}
		if db.GetNonce(addr) != account.Nonce {
			return fmt.Errorf("Failed to validate nonce for account %v\n"+
				"\twant %v\n"+
				"\thave %v",
				addr.Hex(), account.Nonce, db.GetNonce(addr))
		}
		// GetCode not implemented
		// if  db.GetCode(addr) != account.GetCode() {
		// 	log.Fatalf("Failed to validate code for account %v", addr.Hex())
		// }
		for key, value := range account.Storage {
			if db.GetState(addr, key) != value {
				return fmt.Errorf("Failed to validate storage for account %v, key %v\n"+
					"\twant %v\n"+
					"\thave %v",
					addr.Hex(), key.Hex(), value.Hex(), db.GetState(addr, key).Hex())
			}
		}

	}
	return nil
}

// compareSubstateStorage compares an output substate of a transaction to a
// substate genereated from stateDB by the trace replayer
func compareSubstateStorage(record, replay substate.SubstateAlloc) error {
	for addr, recAcc := range record {
		// addr exists in both substate
		if repAcc, exist := replay[addr]; exist {
			for k, xv := range recAcc.Storage {
				// mismatched value or key dones't exist
				if yv, exist := repAcc.Storage[k]; !exist || xv != yv {
					return fmt.Errorf("Error: mismatched value for account %v, key %v\n"+
						"\twant %v\n"+
						"\thave %v",
						addr, k, xv, yv)
				}
			}
			for k, yv := range repAcc.Storage {
				// key exists when expecting nil
				if xv, exist := recAcc.Storage[k]; !exist {
					return fmt.Errorf("Error: mismatched value for account %v, key %v\n"+
						"\twant %v\n"+
						"\thave %v",
						addr, k, xv, yv)
				}
			}
		} else {
			if len(recAcc.Storage) > 0 {
				return fmt.Errorf("Error: addr %v doesn't exist\n", addr)
			}
			//else ignores address which has no storage
		}
	}

	// checks for unexpected address in replayed substate
	for addr := range replay {
		if _, exist := record[addr]; !exist {
			return fmt.Errorf("Error: unexpected address %v\n", addr)
		}
	}
	return nil
}

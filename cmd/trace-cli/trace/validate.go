package trace

import (
	"bytes"
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/state"
	"github.com/ethereum/go-ethereum/substate"
)

// validateStateDB validates whether the world-state is contained in the db object.
// NB: We can only check what must be in the db (but cannot check whether db stores more).
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
		if bytes.Compare(db.GetCode(addr), account.Code) != 0 {
			return fmt.Errorf("Failed to validate code for account %v", addr.Hex())
		}
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

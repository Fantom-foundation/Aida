package validator

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// validateWorldState compares states of accounts in stateDB to an expected set of states.
// If fullState mode, check if expected state is contained in stateDB.
// If partialState mode, check for equality of sets.
func validateWorldState(cfg *utils.Config, db state.VmStateDB, expectedAlloc txcontext.WorldState, log logger.Logger) error {
	var err error
	switch cfg.StateValidationMode {
	case utils.SubsetCheck:
		err = doSubsetValidation(expectedAlloc, db, cfg.UpdateOnFailure)
	case utils.EqualityCheck:
		vmAlloc := db.GetSubstatePostAlloc()
		isEqual := expectedAlloc.Equal(vmAlloc)
		if !isEqual {
			err = fmt.Errorf("inconsistent output: alloc")
			printAllocationDiffSummary(expectedAlloc, vmAlloc, log)

			return err
		}
	}
	return err
}

// printIfDifferent compares two values of any types and reports differences if any.
func printIfDifferent[T comparable](label string, want, have T, log logger.Logger) bool {
	if want != have {
		log.Errorf("Different %s:\nwant: %v\nhave: %v\n", label, want, have)
		return true
	}
	return false
}

// printIfDifferentBytes compares two values of byte type and reports differences if any.
func printIfDifferentBytes(label string, want, have []byte, log logger.Logger) bool {
	if !bytes.Equal(want, have) {
		log.Errorf("Different %s:\nwant: %v\nhave: %v\n", label, want, have)
		return true
	}
	return false
}

// printIfDifferentBigInt compares two values of big int type and reports differences if any.
func printIfDifferentBigInt(label string, want, have *big.Int, log logger.Logger) bool {
	if want == nil && have == nil {
		return false
	}
	if want == nil || have == nil || want.Cmp(have) != 0 {
		log.Errorf("Different %s:\nwant: %v\nhave: %v\n", label, want, have)
		return true
	}
	return false
}

// printLogDiffSummary compares two tx logs and reports differences if any.
func printLogDiffSummary(label string, want, have *types.Log, log logger.Logger) {
	printIfDifferent(fmt.Sprintf("%s.address", label), want.Address, have.Address, log)
	if !printIfDifferent(fmt.Sprintf("%s.Topics size", label), len(want.Topics), len(have.Topics), log) {
		for i := range want.Topics {
			printIfDifferent(fmt.Sprintf("%s.Topics[%d]", label, i), want.Topics[i], have.Topics[i], log)
		}
	}
	printIfDifferentBytes(fmt.Sprintf("%s.data", label), want.Data, have.Data, log)
}

// printAllocationDiffSummary compares atrributes and existence of accounts and reports differences if any.
func printAllocationDiffSummary(want, have txcontext.WorldState, log logger.Logger) {
	printIfDifferent("substate alloc size", want.Len(), have.Len(), log)

	want.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
		if have.Get(addr) == nil {
			log.Errorf("\tmissing address=%v\n", addr)
		}
	})

	have.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
		if want.Get(addr) == nil {
			log.Errorf("\textra address=%v\n", addr)
		}
	})

	have.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
		wantAcc := want.Get(addr)
		printAccountDiffSummary(fmt.Sprintf("key=%v:", addr), wantAcc, acc, log)
	})

}

// PrintAccountDiffSummary compares attributes of two accounts and reports differences if any.
func printAccountDiffSummary(label string, want, have txcontext.Account, log logger.Logger) {
	printIfDifferent(fmt.Sprintf("%s.Nonce", label), want.GetNonce(), have.GetNonce(), log)
	printIfDifferentBigInt(fmt.Sprintf("%s.Balance", label), want.GetBalance(), have.GetBalance(), log)
	printIfDifferentBytes(fmt.Sprintf("%s.Code", label), want.GetCode(), have.GetCode(), log)

	printIfDifferent(fmt.Sprintf("len(%s.Storage)", label), want.GetStorageSize(), have.GetStorageSize(), log)

	want.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
		haveValueHash := have.GetStorageAt(keyHash)
		if haveValueHash != valueHash {
			log.Errorf("\t%s.Storage misses key %v val %v\n", label, keyHash, valueHash)
		}
	})

	have.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
		wantValueHash := want.GetStorageAt(keyHash)
		if wantValueHash != valueHash {
			log.Errorf("\t%s.Storage has extra key %v\n", label, keyHash)
		}
	})

	have.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
		wantValueHash := want.GetStorageAt(keyHash)
		printIfDifferent(fmt.Sprintf("%s.Storage[%v]", label, keyHash), wantValueHash, valueHash, log)
	})

}

// doSubsetValidation validates whether the given alloc is contained in the db object.
// NB: We can only check what must be in the db (but cannot check whether db stores more).
func doSubsetValidation(alloc txcontext.WorldState, db state.VmStateDB, updateOnFail bool) error {
	var err string

	alloc.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
		if !db.Exist(addr) {
			err += fmt.Sprintf("  Account %v does not exist\n", addr.Hex())
			if updateOnFail {
				db.CreateAccount(addr)
			}
		}
		accBalance := acc.GetBalance()

		if balance := db.GetBalance(addr); accBalance.Cmp(balance) != 0 {
			err += fmt.Sprintf("  Failed to validate balance for account %v\n"+
				"    have %v\n"+
				"    want %v\n",
				addr.Hex(), balance, accBalance)
			if updateOnFail {
				db.SubBalance(addr, balance)
				db.AddBalance(addr, accBalance)
			}
		}
		if nonce := db.GetNonce(addr); nonce != acc.GetNonce() {
			err += fmt.Sprintf("  Failed to validate nonce for account %v\n"+
				"    have %v\n"+
				"    want %v\n",
				addr.Hex(), nonce, acc.GetNonce())
			if updateOnFail {
				db.SetNonce(addr, acc.GetNonce())
			}
		}
		if code := db.GetCode(addr); bytes.Compare(code, acc.GetCode()) != 0 {
			err += fmt.Sprintf("  Failed to validate code for account %v\n"+
				"    have len %v\n"+
				"    want len %v\n",
				addr.Hex(), len(code), len(acc.GetCode()))
			if updateOnFail {
				db.SetCode(addr, acc.GetCode())
			}
		}

		// validate Storage
		acc.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
			if db.GetState(addr, keyHash) != valueHash {
				err += fmt.Sprintf("  Failed to validate storage for account %v, key %v\n"+
					"    have %v\n"+
					"    want %v\n",
					addr.Hex(), keyHash.Hex(), db.GetState(addr, keyHash).Hex(), valueHash.Hex())
				if updateOnFail {
					db.SetState(addr, keyHash, valueHash)
				}
			}
		})

	})

	if len(err) > 0 {
		return fmt.Errorf(err)
	}
	return nil
}

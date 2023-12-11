package validator

import (
	"bytes"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
)

// MakeLiveDbValidator creates an extension which validates LIVE StateDb
func MakeLiveDbValidator(cfg *utils.Config) executor.Extension[*substate.Substate] {
	if !cfg.ValidateTxState {
		return extension.NilExtension[*substate.Substate]{}
	}

	log := logger.NewLogger(cfg.LogLevel, "Tx-Verifier")

	return makeLiveDbValidator(cfg, log)
}

func makeLiveDbValidator(cfg *utils.Config, log logger.Logger) *liveDbTxValidator {
	return &liveDbTxValidator{
		makeStateDbValidator(cfg, log),
	}
}

type liveDbTxValidator struct {
	*stateDbValidator
}

// PreTransaction validates InputAlloc in given substate
func (v *liveDbTxValidator) PreTransaction(state executor.State[*substate.Substate], ctx *executor.Context) error {
	err := validateSubstateAlloc(ctx.State, state.Data.InputAlloc, v.cfg)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("live-db-validator err:\nblock %v tx %v\n input alloc is not contained in the state-db\n %v\n", state.Block, state.Transaction, err)

	if v.isErrFatal(err, ctx.ErrorInput) {
		return err
	}

	return nil
}

// PostTransaction validates OutputAlloc in given substate
func (v *liveDbTxValidator) PostTransaction(state executor.State[*substate.Substate], ctx *executor.Context) error {
	err := validateSubstateAlloc(ctx.State, state.Data.OutputAlloc, v.cfg)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("live-db-validator err:\noutput error at block %v tx %v; %v", state.Block, state.Transaction, err)

	if v.isErrFatal(err, ctx.ErrorInput) {
		return err
	}

	return nil
}

// MakeArchiveDbValidator creates an extension which validates ARCHIVE StateDb
func MakeArchiveDbValidator(cfg *utils.Config) executor.Extension[*substate.Substate] {
	if !cfg.ValidateTxState {
		return extension.NilExtension[*substate.Substate]{}
	}

	log := logger.NewLogger(cfg.LogLevel, "Tx-Verifier")

	return makeArchiveDbValidator(cfg, log)
}

func makeArchiveDbValidator(cfg *utils.Config, log logger.Logger) *archiveDbValidator {
	return &archiveDbValidator{
		makeStateDbValidator(cfg, log),
	}
}

type archiveDbValidator struct {
	*stateDbValidator
}

// PreTransaction validates InputAlloc in given substate
func (v *archiveDbValidator) PreTransaction(state executor.State[*substate.Substate], ctx *executor.Context) error {
	err := validateSubstateAlloc(ctx.Archive, state.Data.InputAlloc, v.cfg)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("archive-db-validator err:\nblock %v tx %v\n input alloc is not contained in the state-db\n %v\n", state.Block, state.Transaction, err)

	if v.isErrFatal(err, ctx.ErrorInput) {
		return err
	}

	return nil
}

// PostTransaction validates VmAlloc
func (v *archiveDbValidator) PostTransaction(state executor.State[*substate.Substate], ctx *executor.Context) error {
	err := validateSubstateAlloc(ctx.Archive, state.Data.OutputAlloc, v.cfg)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("archive-db-validator err:\noutput error at block %v tx %v; %v", state.Block, state.Transaction, err)

	if v.isErrFatal(err, ctx.ErrorInput) {
		return err
	}

	return nil
}

// makeStateDbValidator creates an extension that validates StateDb.
// stateDbValidator should always be inherited depending on what
// type of StateDb we are working with
func makeStateDbValidator(cfg *utils.Config, log logger.Logger) *stateDbValidator {
	return &stateDbValidator{
		cfg:            cfg,
		log:            log,
		numberOfErrors: new(atomic.Int32),
	}
}

type stateDbValidator struct {
	extension.NilExtension[*substate.Substate]
	cfg            *utils.Config
	log            logger.Logger
	numberOfErrors *atomic.Int32
}

// PreRun informs the user that stateDbValidator is enabled and that they should expect slower processing speed.
func (v *stateDbValidator) PreRun(executor.State[*substate.Substate], *executor.Context) error {
	v.log.Warning("Transaction verification is enabled, this may slow down the block processing.")

	if v.cfg.ContinueOnFailure {
		v.log.Warningf("Continue on Failure for transaction validation is enabled, yet "+
			"block processing will stop after %v encountered issues. (0 is endless)", v.cfg.MaxNumErrors)
	}

	return nil
}

// isErrFatal decides whether given error should stop the program or not depending on ContinueOnFailure and MaxNumErrors.
func (v *stateDbValidator) isErrFatal(err error, ch chan error) bool {
	// ContinueOnFailure is disabled, return the error and exit the program
	if !v.cfg.ContinueOnFailure {
		return true
	}

	ch <- err
	v.numberOfErrors.Add(1)

	// endless run
	if v.cfg.MaxNumErrors == 0 {
		return false
	}

	// too many errors
	if int(v.numberOfErrors.Load()) >= v.cfg.MaxNumErrors {
		return true
	}

	return false
}

// validateSubstateAlloc compares states of accounts in stateDB to an expected set of states.
// If fullState mode, check if expected state is contained in stateDB.
// If partialState mode, check for equality of sets.
func validateSubstateAlloc(db state.VmStateDB, expectedAlloc substate.SubstateAlloc, cfg *utils.Config) error {
	var err error
	switch cfg.StateValidationMode {
	case utils.SubsetCheck:
		err = doSubsetValidation(expectedAlloc, db, cfg.UpdateOnFailure)
	case utils.EqualityCheck:
		vmAlloc := db.GetSubstatePostAlloc()
		isEqual := expectedAlloc.Equal(vmAlloc)
		if !isEqual {
			// TODO bring back original functionality of PrintAllocationDiffSummary
			err = fmt.Errorf("inconsistent output: alloc")
			for address, vmAccount := range vmAlloc {
				// get account from expectedAlloc
				account := expectedAlloc[address]
				if account == nil {
					err = errors.Join(err, fmt.Errorf("account %v does not exist", address.Hex()))
					continue
				}
				// compare account fields
				if account.Nonce != vmAccount.Nonce {
					err = errors.Join(err, fmt.Errorf("account %v nonce mismatch: have %v, want %v", address.Hex(), vmAccount.Nonce, account.Nonce))
				}
				if account.Balance.Cmp(vmAccount.Balance) != 0 {
					err = errors.Join(err, fmt.Errorf("account %v balance mismatch: have %v, want %v", address.Hex(), vmAccount.Balance, account.Balance))
				}
				if bytes.Compare(account.Code, vmAccount.Code) != 0 {
					err = errors.Join(err, fmt.Errorf("account %v code mismatch: have %v, want %v", address.Hex(), vmAccount.Code, account.Code))
				}
				// compare storage
				for key, value := range account.Storage {
					if vmAccount.Storage[key] != value {
						err = errors.Join(err, fmt.Errorf("account %v storage mismatch: have %v, want %v", address.Hex(), vmAccount.Storage[key], value))
					}
				}
			}

			return err
		}
	}
	return err
}

// doSubsetValidation validates whether the given alloc is contained in the db object.
// NB: We can only check what must be in the db (but cannot check whether db stores more).
func doSubsetValidation(alloc substate.SubstateAlloc, db state.VmStateDB, updateOnFail bool) error {
	var err string
	for addr, account := range alloc {
		if !db.Exist(addr) {
			err += fmt.Sprintf("  Account %v does not exist\n", addr.Hex())
			if updateOnFail {
				db.CreateAccount(addr)
			}
		}
		if balance := db.GetBalance(addr); account.Balance.Cmp(balance) != 0 {
			err += fmt.Sprintf("  Failed to validate balance for account %v\n"+
				"    have %v\n"+
				"    want %v\n",
				addr.Hex(), balance, account.Balance)
			if updateOnFail {
				db.SubBalance(addr, balance)
				db.AddBalance(addr, account.Balance)
			}
		}
		if nonce := db.GetNonce(addr); nonce != account.Nonce {
			err += fmt.Sprintf("  Failed to validate nonce for account %v\n"+
				"    have %v\n"+
				"    want %v\n",
				addr.Hex(), nonce, account.Nonce)
			if updateOnFail {
				db.SetNonce(addr, account.Nonce)
			}
		}
		if code := db.GetCode(addr); bytes.Compare(code, account.Code) != 0 {
			err += fmt.Sprintf("  Failed to validate code for account %v\n"+
				"    have len %v\n"+
				"    want len %v\n",
				addr.Hex(), len(code), len(account.Code))
			if updateOnFail {
				db.SetCode(addr, account.Code)
			}
		}
		for key, value := range account.Storage {
			if db.GetState(addr, key) != value {
				err += fmt.Sprintf("  Failed to validate storage for account %v, key %v\n"+
					"    have %v\n"+
					"    want %v\n",
					addr.Hex(), key.Hex(), db.GetState(addr, key).Hex(), value.Hex())
				if updateOnFail {
					db.SetState(addr, key, value)
				}
			}
		}
	}
	if len(err) > 0 {
		return fmt.Errorf(err)
	}
	return nil
}

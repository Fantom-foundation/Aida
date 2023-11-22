package validator

import (
	"bytes"
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
)

// MakeLiveDbTxValidator creates an extension which validates LIVE StateDb
func MakeLiveDbTxValidator(cfg *utils.Config) executor.Extension[*substate.Substate] {
	if !cfg.ValidateTxState {
		return extension.NilExtension[*substate.Substate]{}
	}

	log := logger.NewLogger(cfg.LogLevel, "Tx-Verifier")

	return makeLiveDbTxValidator(cfg, log)
}

func makeLiveDbTxValidator(cfg *utils.Config, log logger.Logger) *liveDbTxValidator {
	return &liveDbTxValidator{
		makeTxValidator(cfg, log),
	}
}

type liveDbTxValidator struct {
	*txValidator
}

// PreTransaction validates InputAlloc in given substate
func (v *liveDbTxValidator) PreTransaction(state executor.State[*substate.Substate], ctx *executor.Context) error {
	err := validateStateDb(state.Data.InputAlloc, ctx.State, v.cfg.UpdateOnFailure)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("block %v tx %v\n input alloc is not contained in the state-db\n %v\n", state.Block, state.Transaction, err)

	if v.isErrFatal(err, ctx.ErrorInput) {
		return err
	}

	return nil
}

// PostTransaction validates OutputAlloc in given substate
func (v *liveDbTxValidator) PostTransaction(state executor.State[*substate.Substate], ctx *executor.Context) error {
	err := validateVmAlloc(ctx.State, state.Data.OutputAlloc, v.cfg)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("output error at block %v tx %v; %v", state.Block, state.Transaction, err)

	if v.isErrFatal(err, ctx.ErrorInput) {
		return err
	}

	return nil
}

// MakeArchiveDbTxValidator creates an extension which validates ARCHIVE StateDb
func MakeArchiveDbTxValidator(cfg *utils.Config) executor.Extension[*substate.Substate] {
	if !cfg.ValidateTxState {
		return extension.NilExtension[*substate.Substate]{}
	}

	log := logger.NewLogger(cfg.LogLevel, "Tx-Verifier")

	return makeArchiveDbTxValidator(cfg, log)
}

func makeArchiveDbTxValidator(cfg *utils.Config, log logger.Logger) *archiveDbTxValidator {
	return &archiveDbTxValidator{
		makeTxValidator(cfg, log),
	}
}

type archiveDbTxValidator struct {
	*txValidator
}

// PreTransaction validates InputAlloc in given substate
func (v *archiveDbTxValidator) PreTransaction(state executor.State[*substate.Substate], ctx *executor.Context) error {
	err := validateStateDb(state.Data.InputAlloc, ctx.Archive, v.cfg.UpdateOnFailure)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("block %v tx %v\n input alloc is not contained in the state-db\n %v\n", state.Block, state.Transaction, err)

	if v.isErrFatal(err, ctx.ErrorInput) {
		return err
	}

	return nil
}

// PostTransaction validates OutputAlloc in given substate
func (v *archiveDbTxValidator) PostTransaction(state executor.State[*substate.Substate], ctx *executor.Context) error {
	err := validateVmAlloc(ctx.Archive, state.Data.OutputAlloc, v.cfg)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("output error at block %v tx %v; %v", state.Block, state.Transaction, err)

	if v.isErrFatal(err, ctx.ErrorInput) {
		return err
	}

	return nil
}

// makeTxValidator creates an extension that validates StateDb.
// txValidator should always be inherited depending on what
// type of StateDb we are working with
func makeTxValidator(cfg *utils.Config, log logger.Logger) *txValidator {
	return &txValidator{
		cfg: cfg,
		log: log,
	}
}

type txValidator struct {
	extension.NilExtension[*substate.Substate]
	cfg            *utils.Config
	log            logger.Logger
	numberOfErrors int
}

// PreRun informs the user that txValidator is enabled and that they should expect slower processing speed.
func (v *txValidator) PreRun(executor.State[*substate.Substate], *executor.Context) error {
	v.log.Warning("Transaction verification is enabled, this may slow down the block processing.")

	if v.cfg.ContinueOnFailure {
		v.log.Warningf("Continue on Failure for transaction validation is enabled, yet "+
			"block processing will stop after %v encountered issues.", v.cfg.MaxNumErrors)
	}

	return nil
}

// isErrFatal decides whether given error should stop the program or not depending on ContinueOnFailure and MaxNumErrors.
func (v *txValidator) isErrFatal(err error, ch chan error) bool {
	// ContinueOnFailure is disabled, return the error and exit the program
	if !v.cfg.ContinueOnFailure {
		return true
	}

	ch <- err
	v.numberOfErrors++

	// endless run
	if v.cfg.MaxNumErrors == 0 {
		return false
	}

	// too many errors
	if v.numberOfErrors >= v.cfg.MaxNumErrors {
		return true
	}

	return false
}

// validateStateDb validates whether the world-state is contained in the db object.
// NB: We can only check what must be in the db (but cannot check whether db stores more).
func validateStateDb(ws substate.SubstateAlloc, db state.VmStateDB, updateOnFail bool) error {
	var err string
	for addr, account := range ws {
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

// validateVmAlloc compares states of accounts in stateDB to an expected set of states.
// If fullState mode, check if expected state is contained in stateDB.
// If partialState mode, check for equality of sets.
func validateVmAlloc(db state.VmStateDB, expectedAlloc substate.SubstateAlloc, cfg *utils.Config) error {
	var err error
	switch cfg.StateValidationMode {
	case utils.SubsetCheck:
		err = validateStateDb(expectedAlloc, db, !cfg.UpdateOnFailure)
	case utils.EqualityCheck:
		vmAlloc := db.GetSubstatePostAlloc()
		isEqual := expectedAlloc.Equal(vmAlloc)
		if !isEqual {
			err = fmt.Errorf("inconsistent output: alloc")
		}
	}
	return err
}

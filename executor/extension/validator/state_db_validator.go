package validator

import (
	"bytes"
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// MakeLiveDbValidator creates an extension which validates LIVE StateDb
func MakeLiveDbValidator(cfg *utils.Config) executor.Extension[executor.TransactionData] {
	if !cfg.ValidateTxState {
		return extension.NilExtension[executor.TransactionData]{}
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
func (v *liveDbTxValidator) PreTransaction(state executor.State[executor.TransactionData], ctx *executor.Context) error {
	err := v.validateSubstateAlloc(ctx.State, state.Data.GetInputAlloc())
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
func (v *liveDbTxValidator) PostTransaction(state executor.State[executor.TransactionData], ctx *executor.Context) error {
	err := v.validateSubstateAlloc(ctx.State, state.Data.GetOutputAlloc())
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
func MakeArchiveDbValidator(cfg *utils.Config) executor.Extension[executor.TransactionData] {
	if !cfg.ValidateTxState {
		return extension.NilExtension[executor.TransactionData]{}
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
func (v *archiveDbValidator) PreTransaction(state executor.State[executor.TransactionData], ctx *executor.Context) error {
	err := v.validateSubstateAlloc(ctx.Archive, state.Data.GetInputAlloc())
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
func (v *archiveDbValidator) PostTransaction(state executor.State[executor.TransactionData], ctx *executor.Context) error {
	err := v.validateSubstateAlloc(ctx.Archive, state.Data.GetOutputAlloc())
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
	extension.NilExtension[executor.TransactionData]
	cfg            *utils.Config
	log            logger.Logger
	numberOfErrors *atomic.Int32
}

// PreRun informs the user that stateDbValidator is enabled and that they should expect slower processing speed.
func (v *stateDbValidator) PreRun(executor.State[executor.TransactionData], *executor.Context) error {
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
func (v *stateDbValidator) validateSubstateAlloc(db state.VmStateDB, expectedAlloc substate.SubstateAlloc) error {
	var err error
	switch v.cfg.StateValidationMode {
	case utils.SubsetCheck:
		err = doSubsetValidation(expectedAlloc, db, v.cfg.UpdateOnFailure)
	case utils.EqualityCheck:
		vmAlloc := db.GetSubstatePostAlloc()
		isEqual := expectedAlloc.Equal(vmAlloc)
		if !isEqual {
			err = fmt.Errorf("inconsistent output: alloc")
			v.printAllocationDiffSummary(&expectedAlloc, &vmAlloc)

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
func (v *stateDbValidator) printIfDifferentBytes(label string, want, have []byte) bool {
	if !bytes.Equal(want, have) {
		v.log.Errorf("Different %s:\nwant: %v\nhave: %v\n", label, want, have)
		return true
	}
	return false
}

// printIfDifferentBigInt compares two values of big int type and reports differences if any.
func (v *stateDbValidator) printIfDifferentBigInt(label string, want, have *big.Int) bool {
	if want == nil && have == nil {
		return false
	}
	if want == nil || have == nil || want.Cmp(have) != 0 {
		v.log.Errorf("Different %s:\nwant: %v\nhave: %v\n", label, want, have)
		return true
	}
	return false
}

// printLogDiffSummary compares two tx logs and reports differences if any.
func (v *stateDbValidator) printLogDiffSummary(label string, want, have *types.Log) {
	printIfDifferent(fmt.Sprintf("%s.address", label), want.Address, have.Address, v.log)
	if !printIfDifferent(fmt.Sprintf("%s.Topics size", label), len(want.Topics), len(have.Topics), v.log) {
		for i := range want.Topics {
			printIfDifferent(fmt.Sprintf("%s.Topics[%d]", label, i), want.Topics[i], have.Topics[i], v.log)
		}
	}
	v.printIfDifferentBytes(fmt.Sprintf("%s.data", label), want.Data, have.Data)
}

// printAllocationDiffSummary compares atrributes and existence of accounts and reports differences if any.
func (v *stateDbValidator) printAllocationDiffSummary(want, have *substate.SubstateAlloc) {
	printIfDifferent("substate alloc size", len(*want), len(*have), v.log)
	for key := range *want {
		_, present := (*have)[key]
		if !present {
			v.log.Errorf("\tmissing key=%v\n", key)

		}
	}

	for key := range *have {
		_, present := (*want)[key]
		if !present {
			v.log.Errorf("\textra key=%v\n", key)
		}
	}

	for key, is := range *have {
		should, present := (*want)[key]
		if present {
			v.printAccountDiffSummary(fmt.Sprintf("key=%v:", key), should, is)
		}
	}
}

// PrintAccountDiffSummary compares attributes of two accounts and reports differences if any.
func (v *stateDbValidator) printAccountDiffSummary(label string, want, have *substate.SubstateAccount) {
	printIfDifferent(fmt.Sprintf("%s.Nonce", label), want.Nonce, have.Nonce, v.log)
	v.printIfDifferentBigInt(fmt.Sprintf("%s.Balance", label), want.Balance, have.Balance)
	v.printIfDifferentBytes(fmt.Sprintf("%s.Code", label), want.Code, have.Code)

	printIfDifferent(fmt.Sprintf("len(%s.Storage)", label), len(want.Storage), len(have.Storage), v.log)
	for key, val := range want.Storage {
		_, present := have.Storage[key]
		if !present && (val != common.Hash{}) {
			v.log.Errorf("\t%s.Storage misses key %v val %v\n", label, key, val)
		}
	}

	for key := range have.Storage {
		_, present := want.Storage[key]
		if !present {
			v.log.Errorf("\t%s.Storage has extra key %v\n", label, key)
		}
	}

	for key, is := range have.Storage {
		should, present := want.Storage[key]
		if present {
			printIfDifferent(fmt.Sprintf("%s.Storage[%v]", label, key), should, is, v.log)
		}
	}
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

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
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// MakeLiveDbValidator creates an extension which validates LIVE StateDb
func MakeLiveDbValidator(cfg *utils.Config, target ValidateTxTarget) executor.Extension[txcontext.TxContext] {
	if !cfg.ValidateTxState {
		return extension.NilExtension[txcontext.TxContext]{}
	}

	log := logger.NewLogger(cfg.LogLevel, "Tx-Verifier")

	return makeLiveDbValidator(cfg, log, target)
}

func makeLiveDbValidator(cfg *utils.Config, log logger.Logger, target ValidateTxTarget) *liveDbTxValidator {
	return &liveDbTxValidator{
		makeStateDbValidator(cfg, log, target),
	}
}

type liveDbTxValidator struct {
	*stateDbValidator
}

// PreTransaction validates InputAlloc in given substate
func (v *liveDbTxValidator) PreTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	return v.runPreTxValidation("live-db-validator", ctx.State, state, ctx.ErrorInput)
}

// PostTransaction validates OutputAlloc in given substate
func (v *liveDbTxValidator) PostTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	return v.runPostTxValidation("live-db-validator", ctx.State, state, ctx.ExecutionResult, ctx.ErrorInput)
}

// MakeArchiveDbValidator creates an extension which validates ARCHIVE StateDb
func MakeArchiveDbValidator(cfg *utils.Config, target ValidateTxTarget) executor.Extension[txcontext.TxContext] {
	if !cfg.ValidateTxState {
		return extension.NilExtension[txcontext.TxContext]{}
	}

	log := logger.NewLogger(cfg.LogLevel, "Tx-Verifier")

	return makeArchiveDbValidator(cfg, log, target)
}

func makeArchiveDbValidator(cfg *utils.Config, log logger.Logger, target ValidateTxTarget) *archiveDbValidator {
	return &archiveDbValidator{
		makeStateDbValidator(cfg, log, target),
	}
}

type archiveDbValidator struct {
	*stateDbValidator
}

// PreTransaction validates the input WorldState before transaction is executed.
func (v *archiveDbValidator) PreTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	return v.runPreTxValidation("archive-db-validator", ctx.Archive, state, ctx.ErrorInput)
}

// PostTransaction validates the resulting WorldState after transaction is executed.
func (v *archiveDbValidator) PostTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	return v.runPostTxValidation("archive-db-validator", ctx.Archive, state, ctx.ExecutionResult, ctx.ErrorInput)
}

// makeStateDbValidator creates an extension that validates StateDb.
// stateDbValidator should always be inherited depending on what
// type of StateDb we are working with
func makeStateDbValidator(cfg *utils.Config, log logger.Logger, target ValidateTxTarget) *stateDbValidator {
	return &stateDbValidator{
		cfg:            cfg,
		log:            log,
		numberOfErrors: new(atomic.Int32),
		target:         target,
	}
}

type stateDbValidator struct {
	extension.NilExtension[txcontext.TxContext]
	cfg            *utils.Config
	log            logger.Logger
	numberOfErrors *atomic.Int32
	target         ValidateTxTarget
}

// ValidateTxTarget serves for the validator to determine what type of validation to run
type ValidateTxTarget struct {
	WorldState bool // WorldState will validate StateDb PostAlloc
	Receipt    bool // Receipt will validate the TxReceipt after
}

// PreRun informs the user that stateDbValidator is enabled and that they should expect slower processing speed.
func (v *stateDbValidator) PreRun(executor.State[txcontext.TxContext], *executor.Context) error {
	v.log.Warning("Transaction verification is enabled, this may slow down the block processing.")

	if v.cfg.ContinueOnFailure {
		v.log.Warningf("Continue on Failure for transaction validation is enabled, yet "+
			"block processing will stop after %v encountered issues. (0 is endless)", v.cfg.MaxNumErrors)
	}

	return nil
}

func (v *stateDbValidator) runPreTxValidation(tool string, db state.VmStateDB, state executor.State[txcontext.TxContext], errOutput chan error) error {
	if !v.target.WorldState {
		return nil
	}

	err := v.validateWorldState(db, state.Data.GetInputState())
	if err == nil {
		return nil
	}

	err = fmt.Errorf("%v err:\nblock %v tx %v\n world-state input is not contained in the state-db\n %v\n", tool, state.Block, state.Transaction, err)

	if v.isErrFatal(err, errOutput) {
		return err
	}

	return nil
}

func (v *stateDbValidator) runPostTxValidation(tool string, db state.VmStateDB, state executor.State[txcontext.TxContext], res txcontext.Receipt, errOutput chan error) error {
	if v.target.WorldState {
		if err := v.validateWorldState(db, state.Data.GetOutputState()); err != nil {
			err = fmt.Errorf("%v err:\nworld-state output error at block %v tx %v; %v", tool, state.Block, state.Transaction, err)
			if v.isErrFatal(err, errOutput) {
				return err
			}
		}
	}

	if v.target.Receipt {
		if err := v.validateReceipt(res, state.Data.GetReceipt()); err != nil {
			err = fmt.Errorf("%v err:\nvm-result error at block %v tx %v; %v", tool, state.Block, state.Transaction, err)
			if v.isErrFatal(err, errOutput) {
				return err
			}
		}
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

// validateWorldState compares states of accounts in stateDB to an expected set of states.
// If fullState mode, check if expected state is contained in stateDB.
// If partialState mode, check for equality of sets.
func (v *stateDbValidator) validateWorldState(db state.VmStateDB, expectedAlloc txcontext.WorldState) error {
	var err error
	switch v.cfg.StateValidationMode {
	case utils.SubsetCheck:
		err = doSubsetValidation(expectedAlloc, db, v.cfg.UpdateOnFailure)
	case utils.EqualityCheck:
		vmAlloc := db.GetSubstatePostAlloc()
		isEqual := expectedAlloc.Equal(vmAlloc)
		if !isEqual {
			err = fmt.Errorf("inconsistent output: alloc")
			v.printAllocationDiffSummary(expectedAlloc, vmAlloc)

			return err
		}
	}
	return err
}

// validateReceipt compares result from vm against the expected one.
// Error is returned if any mismatch is found.
func (v *stateDbValidator) validateReceipt(got, want txcontext.Receipt) error {
	if !got.Equal(want) {
		return fmt.Errorf(
			"\ngot:\n"+
				"\tstatus: %v\n"+
				"\tbloom: %v\n"+
				"\tlogs: %v\n"+
				"\tcontract address: %v\n"+
				"\tgas used: %v\n"+
				"\nwant:\n"+
				"\tstatus: %v\n"+
				"\tbloom: %v\n"+
				"\tlogs: %v\n"+
				"\tcontract address: %v\n"+
				"\tgas used: %v\n",
			got.GetStatus(),
			got.GetBloom().Big().Uint64(),
			got.GetLogs(),
			got.GetContractAddress(),
			got.GetGasUsed(),
			want.GetStatus(),
			want.GetBloom().Big().Uint64(),
			want.GetLogs(),
			want.GetContractAddress(),
			want.GetGasUsed())
	}

	return nil
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
func (v *stateDbValidator) printAllocationDiffSummary(want, have txcontext.WorldState) {
	printIfDifferent("substate alloc size", want.Len(), have.Len(), v.log)

	want.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
		if have.Get(addr) == nil {
			v.log.Errorf("\tmissing address=%v\n", addr)
		}
	})

	have.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
		if want.Get(addr) == nil {
			v.log.Errorf("\textra address=%v\n", addr)
		}
	})

	have.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
		wantAcc := want.Get(addr)
		v.printAccountDiffSummary(fmt.Sprintf("key=%v:", addr), wantAcc, acc)
	})

}

// PrintAccountDiffSummary compares attributes of two accounts and reports differences if any.
func (v *stateDbValidator) printAccountDiffSummary(label string, want, have txcontext.Account) {
	printIfDifferent(fmt.Sprintf("%s.Nonce", label), want.GetNonce(), have.GetNonce(), v.log)
	v.printIfDifferentBigInt(fmt.Sprintf("%s.Balance", label), want.GetBalance(), have.GetBalance())
	v.printIfDifferentBytes(fmt.Sprintf("%s.Code", label), want.GetCode(), have.GetCode())

	printIfDifferent(fmt.Sprintf("len(%s.Storage)", label), want.GetStorageSize(), have.GetStorageSize(), v.log)

	want.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
		haveValueHash := have.GetStorageAt(keyHash)
		if haveValueHash != valueHash {
			v.log.Errorf("\t%s.Storage misses key %v val %v\n", label, keyHash, valueHash)
		}
	})

	have.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
		wantValueHash := want.GetStorageAt(keyHash)
		if wantValueHash != valueHash {
			v.log.Errorf("\t%s.Storage has extra key %v\n", label, keyHash)
		}
	})

	have.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
		wantValueHash := want.GetStorageAt(keyHash)
		printIfDifferent(fmt.Sprintf("%s.Storage[%v]", label, keyHash), wantValueHash, valueHash, v.log)
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

package validator

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
)

type txValidator struct {
	extension.NilExtension[*substate.Substate]
	cfg    *utils.Config
	log    logger.Logger
	lock   sync.Mutex
	errors []error
}

func MakeTxValidator(cfg *utils.Config) executor.Extension[*substate.Substate] {
	if !cfg.ValidateTxState {
		return extension.NilExtension[*substate.Substate]{}
	}

	log := logger.NewLogger(cfg.LogLevel, "Tx-Verifier")

	return makeTxValidator(cfg, log)
}

func makeTxValidator(cfg *utils.Config, log logger.Logger) *txValidator {
	return &txValidator{
		cfg: cfg,
		log: log,
	}
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

// PreTransaction validates InputAlloc in given substate
func (v *txValidator) PreTransaction(state executor.State[*substate.Substate], ctx *executor.Context) error {
	err := utils.ValidateStateDB(state.Data.InputAlloc, ctx.State, v.cfg.UpdateOnFailure)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("input error at block %v tx %v; %v", state.Block, state.Transaction, err)

	if v.isErrFatal(err) {
		err = errors.New("maximum number of errors occurred")
		v.log.Critical(err)
		return err
	}

	return nil
}

// PostTransaction validates OutputAlloc in given substate
func (v *txValidator) PostTransaction(state executor.State[*substate.Substate], ctx *executor.Context) error {
	err := utils.ValidateStateDB(state.Data.OutputAlloc, ctx.State, v.cfg.UpdateOnFailure)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("output error at block %v tx %v; %v", state.Block, state.Transaction, err)

	if v.isErrFatal(err) {
		err = errors.New("maximum number of errors occurred")
		v.log.Critical(err)
		return err
	}

	return nil
}

// PostRun informs user how many errors were found - if ContinueOnFailureIsEnabled otherwise success is reported.
func (v *txValidator) PostRun(executor.State[*substate.Substate], *executor.Context, error) error {
	v.lock.Lock()
	defer v.lock.Unlock()

	// no errors occurred
	if len(v.errors) == 0 {
		v.log.Noticef("Validation successful!")
		return nil
	}

	v.log.Warningf("%v errors caught", len(v.errors))

	return errors.Join(v.errors...)
}

// isErrFatal decides whether given error should stop the program or not depending on ContinueOnFailure and MaxNumErrors.
func (v *txValidator) isErrFatal(err error) bool {
	v.lock.Lock()
	v.errors = append(v.errors, err)
	v.lock.Unlock()

	// ContinueOnFailure is disabled, return the error thus exit the program
	if !v.cfg.ContinueOnFailure {
		return true
	}

	v.log.Error(err)

	if len(v.errors) >= v.cfg.MaxNumErrors {
		return true
	}

	return false
}

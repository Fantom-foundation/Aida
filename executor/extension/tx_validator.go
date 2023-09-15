package extension

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

type txValidator struct {
	NilExtension
	config *utils.Config
	log    logger.Logger
	lock   sync.Mutex
	errors []error
}

func MakeTxValidator(config *utils.Config) executor.Extension {
	// todo remove this line once utils.ProcessTx(...) is moved here
	// tx validation is integrated in utils.ProcessTx(...) so using the Extension now would validate two times the same substate
	return NilExtension{}

	if !config.ValidateTxState {
		return NilExtension{}
	}

	log := logger.NewLogger(config.LogLevel, "Tx-Verifier")

	return makeTxValidator(config, log)
}

func makeTxValidator(config *utils.Config, log logger.Logger) *txValidator {
	return &txValidator{
		config: config,
		log:    log,
	}
}

// PreRun informs the user that txValidator is enabled and that they should expect slower processing speed.
func (v *txValidator) PreRun(executor.State, *executor.Context) error {

	v.log.Warning("Transaction verification is enabled, this may slow down the block processing.")

	if v.config.ContinueOnFailure {
		v.log.Warningf("Continue on Failure for transaction validation is enabled, yet "+
			"block processing will stop after %v encountered issues.", v.config.MaxNumErrors)
	}

	return nil
}

// PreTransaction validates InputAlloc in given substate
func (v *txValidator) PreTransaction(state executor.State, context *executor.Context) error {
	err := utils.ValidateStateDB(state.Substate.InputAlloc, context.State, v.config.UpdateOnFailure)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("input error at block %v tx %v; %v", state.Block, state.Transaction, err)

	return v.checkTxErr(err)
}

// PostTransaction validates OutputAlloc in given substate
func (v *txValidator) PostTransaction(state executor.State, context *executor.Context) error {
	err := utils.ValidateStateDB(state.Substate.OutputAlloc, context.State, v.config.UpdateOnFailure)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("output error at block %v tx %v; %v", state.Block, state.Transaction, err)

	return v.checkTxErr(err)
}

// PostRun informs user how many errors were found - if ContinueOnFailureIsEnabled otherwise success is reported.
func (v *txValidator) PostRun(executor.State, *executor.Context, error) error {
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

// checkTxErr decides whether return the error (and exit the app) or log and collect it. This decision is based on configuration.
func (v *txValidator) checkTxErr(err error) error {
	// ContinueOnFailure is disabled, return the error thus exit the program
	if !v.config.ContinueOnFailure {
		return err
	}

	// ContinueOnFailure is enabled, log the error to user and collect it
	v.log.Error(err)

	// this func must be thread safe
	v.lock.Lock()
	defer v.lock.Unlock()
	v.errors = append(v.errors, err)

	if len(v.errors) >= v.config.MaxNumErrors {
		return errors.New("maximum number of errors occurred")
	}

	return nil
}

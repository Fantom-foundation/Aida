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

// PreRun informs user the txValidator is enabled thus he should count with slower processing speed.
func (v *txValidator) PreRun(_ executor.State) error {

	v.log.Warning("Transaction verification is enabled, this may slow down the process.")

	if v.config.ContinueOnFailure {
		v.log.Warningf("Continue on Failure for tx validation is enabled though "+
			"program will exit when %v errors occur!", v.config.MaxNumErrors)
	}

	return nil
}

// PreTransaction validates InputAlloc in given substate
func (v *txValidator) PreTransaction(state executor.State) error {
	err := utils.ValidateStateDB(state.Substate.InputAlloc, state.State, v.config.UpdateOnFailure)
	if err == nil {
		return nil
	}

	// this func must be thread safe
	v.lock.Lock()
	defer v.lock.Unlock()

	e := fmt.Errorf("input error at block %v tx %v; %v", state.Block, state.Transaction, err)
	// ContinueOnFailure is disabled, return the error thus exit the program
	if !v.config.ContinueOnFailure {
		return e
	}

	v.log.Error(e)
	// ContinueOnFailure is enabled, log the error to user
	v.errors = append(v.errors, err)

	// check if error cap has been reached
	if len(v.errors) >= v.config.MaxNumErrors {
		return errors.New("maximum number of errors occurred")
	}

	return nil
}

// PostTransaction validates OutputAlloc in given substate
func (v *txValidator) PostTransaction(state executor.State) error {
	err := utils.ValidateStateDB(state.Substate.InputAlloc, state.State, v.config.UpdateOnFailure)
	if err == nil {
		return nil
	}

	// this func must be thread safe
	v.lock.Lock()
	defer v.lock.Unlock()

	e := fmt.Errorf("output error at block %v tx %v; %v", state.Block, state.Transaction, err)
	// ContinueOnFailure is disabled, return the error thus exit the program
	if !v.config.ContinueOnFailure {
		return e
	}

	v.log.Error(e)
	// ContinueOnFailure is enabled, log the error to user
	v.errors = append(v.errors, err)

	// check if error cap has been reached
	if len(v.errors) >= v.config.MaxNumErrors {
		return errors.New("maximum number of errors occurred")
	}

	return nil
}

// PostRun informs user how many errors were found - if ContinueOnFailureIsEnabled otherwise success is reported.
func (v *txValidator) PostRun(_ executor.State, _ error) error {
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

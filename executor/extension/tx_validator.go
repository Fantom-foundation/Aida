package extension

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

const defaultMaxErrors = 50

type txValidator struct {
	NilExtension
	config          *utils.Config
	log             logger.Logger
	updateOnFailure bool
	maxErrors       int
	lock            sync.Mutex
	inputErrors     []error
	outputErrors    []error
}

func MakeTxValidator(config *utils.Config, updateOnFailure bool, maxErrors int) executor.Extension {
	if !config.ValidateTxState {
		return NilExtension{}
	}

	if maxErrors == 0 {
		maxErrors = defaultMaxErrors
	}

	return &txValidator{
		config:          config,
		log:             logger.NewLogger(config.LogLevel, "Tx-Verifier"),
		updateOnFailure: updateOnFailure,
		maxErrors:       maxErrors,
	}
}

// PreRun informs user the txValidator is enabled thus he should count with slower processing speed.
func (v *txValidator) PreRun(_ executor.State) error {

	v.log.Warning("Transaction verification is enabled, this may slow down the process.")

	if v.config.ContinueOnFailure {
		v.log.Warningf("Continue on Failure for tx validation is enabled though "+
			"program will exit when %v errors occur!", v.maxErrors)
	}

	return nil
}

// PreTransaction validates InputAlloc in given substate
func (v *txValidator) PreTransaction(state executor.State) error {
	err := utils.ValidateStateDB(state.Substate.InputAlloc, state.State, v.updateOnFailure)
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
	v.inputErrors = append(v.inputErrors, err)

	// check if error cap has been reached
	if len(v.inputErrors)+len(v.outputErrors) >= v.maxErrors {
		return errors.New("maximum number of errors occurred")
	}

	return nil
}

// PostTransaction validates OutputAlloc in given substate
func (v *txValidator) PostTransaction(state executor.State) error {
	err := utils.ValidateStateDB(state.Substate.InputAlloc, state.State, v.updateOnFailure)
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
	v.inputErrors = append(v.inputErrors, err)

	// check if error cap has been reached
	if len(v.inputErrors)+len(v.outputErrors) >= v.maxErrors {
		return errors.New("maximum number of errors occurred")
	}

	return nil
}

// PostRun informs user how many errors were found - if ContinueOnFailureIsEnabled otherwise success is reported.
func (v *txValidator) PostRun(_ executor.State, _ error) error {
	v.lock.Lock()
	defer v.lock.Unlock()

	// no errors occurred
	if len(v.outputErrors) == 0 && len(v.inputErrors) == 0 {
		v.log.Noticef("Validation successful!")
		return nil
	}

	v.log.Warningf("%v output errors caught", len(v.outputErrors))
	v.log.Warningf("%v input errors caught", len(v.inputErrors))

	var err error

	err = errors.Join(v.inputErrors...)
	err = errors.Join(v.outputErrors...)

	return err
}

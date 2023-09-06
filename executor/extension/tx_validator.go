package extension

import (
	"errors"
	"fmt"

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
	errCounter      int
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
	v.log.Warning("Transaction verification is enabled, this will drastically slow down the process.")

	if v.config.ContinueOnFailure {
		v.log.Warningf("Continue on Failure for tx validation is enabled though "+
			"program will exit when %v errors occur!", v.errCounter)
	}

	return nil
}

// PostTransaction validates StateDB. If any error occurs and ContinueOnFailure is enabled, if
// adds one error to the counter (txValidator.errCounter). If the counter is larger than threshold
// (txValidator.maxErrors), or ContinueOnFailure is disabled, error is returned exiting the program.
func (v *txValidator) PostTransaction(state executor.State) error {
	err := utils.ValidateStateDB(state.Substate.InputAlloc, state.State, v.updateOnFailure)
	if err == nil {
		return nil
	}

	v.errCounter++

	if !v.config.ContinueOnFailure {
		return fmt.Errorf("input alloc is not contained in the state_db; %v", err)
	}

	v.log.Errorf("input alloc is not contained in the state_db; %v", err)

	if v.errCounter >= v.maxErrors {
		return errors.New("maximum number of errors occurred")
	}

	v.log.Warningf("Error number %v", v.errCounter)

	return nil
}

// PostRun informs user how many errors were found - if ContinueOnFailureIsEnabled otherwise success is reported.
func (v *txValidator) PostRun(_ executor.State, _ error) error {
	if v.config.ContinueOnFailure {
		v.log.Noticef("Validation finished! In total %v errors occurred", v.errCounter)
	} else {
		v.log.Noticef("Validation successful!")
	}

	return nil

}

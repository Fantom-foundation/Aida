// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package validator

import (
	"fmt"
	"sync/atomic"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
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
	WorldState bool // validate state before and after processing a transaction
	Receipt    bool // validate content of transaction receipt
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

	err := validateWorldState(v.cfg, db, state.Data.GetInputState(), v.log)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("%v err:\nblock %v tx %v\n world-state input is not contained in the state-db\n %v\n", tool, state.Block, state.Transaction, err)

	if v.isErrFatal(err, errOutput) {
		return err
	}

	return nil
}

func (v *stateDbValidator) runPostTxValidation(tool string, db state.VmStateDB, state executor.State[txcontext.TxContext], res txcontext.Result, errOutput chan error) error {
	if v.target.WorldState {
		if err := validateWorldState(v.cfg, db, state.Data.GetOutputState(), v.log); err != nil {
			err = fmt.Errorf("%v err:\nworld-state output error at block %v tx %v; %v", tool, state.Block, state.Transaction, err)
			if v.isErrFatal(err, errOutput) {
				return err
			}
		}
	}

	// TODO remove state.Transaction < 99999 after patch aida-db
	if v.target.Receipt && state.Transaction < 99999 {
		if err := v.validateReceipt(res.GetReceipt(), state.Data.GetResult().GetReceipt()); err != nil {
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

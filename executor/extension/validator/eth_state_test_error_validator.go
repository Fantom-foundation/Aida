// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package validator

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

func MakeEthStateTestErrorValidator(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	if !cfg.Validate {
		return extension.NilExtension[txcontext.TxContext]{}
	}
	return makeEthStateTestErrorValidator(cfg, logger.NewLogger(cfg.LogLevel, "ethStateTestErrorValidator"))
}

func makeEthStateTestErrorValidator(cfg *utils.Config, log logger.Logger) executor.Extension[txcontext.TxContext] {
	return &ethStateTestErrorValidator{
		cfg: cfg,
		log: log,
	}
}

type ethStateTestErrorValidator struct {
	extension.NilExtension[txcontext.TxContext]
	cfg *utils.Config
	log logger.Logger
}

// PreBlock validates world state.
func (e *ethStateTestErrorValidator) PreBlock(s executor.State[txcontext.TxContext], ctx *executor.Context) error {
	err := validateWorldState(e.cfg, ctx.State, s.Data.GetInputState(), e.log)
	if err != nil {
		return fmt.Errorf("pre alloc validation failed; %v", err)
	}

	return nil
}

// PostBlock validates error returned by the transaction processor.
func (e *ethStateTestErrorValidator) PostBlock(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	var err error
	_, got := ctx.ExecutionResult.GetRawResult()
	_, want := state.Data.GetResult().GetRawResult()
	if want == nil && got == nil {
		return nil
	}
	if got == nil && want != nil {
		err = fmt.Errorf("expected error %w, got no error\ntest-info:\n%s", want, state.Data)
	}
	if got != nil && want == nil {
		err = fmt.Errorf("unexpected error: %w\ntest-info:\n%s", got, state.Data)
	}
	if want != nil && got != nil {
		// TODO check error string - requires somewhat complex string parsing
		return nil
	}

	return e.checkFatality(err, ctx.ErrorInput)
}

func (e *ethStateTestErrorValidator) checkFatality(err error, errChan chan error) error {
	if !e.cfg.ContinueOnFailure {
		return err
	}
	errChan <- err
	return nil
}

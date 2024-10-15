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
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

func MakeEthStateTestLogHashValidator(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	if !cfg.Validate {
		return extension.NilExtension[txcontext.TxContext]{}
	}
	return makeEthStateTestLogHashValidator(cfg)
}

func makeEthStateTestLogHashValidator(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	return &ethStateTestLogHashValidator{
		cfg: cfg,
	}
}

type ethStateTestLogHashValidator struct {
	extension.NilExtension[txcontext.TxContext]
	cfg *utils.Config
}

func (e *ethStateTestLogHashValidator) PostBlock(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	var err error
	if got, want := utils.RlpHash(ctx.ExecutionResult.GetReceipt().GetLogs()), state.Data.GetLogsHash(); got != want {
		err = fmt.Errorf("unexpected logs hash, got %x, want %x", got, want)
		if !e.cfg.ContinueOnFailure {
			return err
		}

		ctx.ErrorInput <- err
	}

	return nil
}

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

func MakeEthStateTestStateHashValidator(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	if !cfg.Validate {
		return extension.NilExtension[txcontext.TxContext]{}
	}
	return makeEthStateTestStateHashValidator(cfg, logger.NewLogger(cfg.LogLevel, "EthStateTestValidator"))
}

func makeEthStateTestStateHashValidator(cfg *utils.Config, log logger.Logger) executor.Extension[txcontext.TxContext] {
	return &ethStateTestStateHashValidator{
		cfg: cfg,
		log: log,
	}
}

type ethStateTestStateHashValidator struct {
	extension.NilExtension[txcontext.TxContext]
	cfg *utils.Config
	log logger.Logger
}

// PostBlock validates state root hash.
// This needs to be done here instead of PostTransaction because EndBlock is being called in PostTransaction in
// executor/extension/statedb/eth_state_test_scope_event_emitter.go, and it needs to be called before GetHash.
func (e *ethStateTestStateHashValidator) PostBlock(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	got, err := ctx.State.GetHash()
	if err != nil {
		return err
	}
	want := state.Data.GetStateHash()
	if got != want {
		err = fmt.Errorf("unexpected root hash, got: %s, want: %s", got, want)
		if !e.cfg.ContinueOnFailure {
			return err
		}
		ctx.ErrorInput <- err
	}

	return nil
}

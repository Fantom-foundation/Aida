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

func MakeShadowDbValidator(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	if cfg.ShadowDb {
		return makeShadowDbValidator(cfg)
	}

	return extension.NilExtension[txcontext.TxContext]{}
}

func makeShadowDbValidator(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	return &shadowDbValidator{
		cfg: cfg,
	}
}

type shadowDbValidator struct {
	extension.NilExtension[txcontext.TxContext]
	cfg *utils.Config
}

func (e *shadowDbValidator) PostTransaction(s executor.State[txcontext.TxContext], ctx *executor.Context) error {
	// Retrieve hash from the state, if this there is mismatch between prime and shadow db error is returned
	got, err := ctx.State.GetHash()
	if err != nil {
		return err
	}

	// Todo move to state_hash_validator in different PR
	want := s.Data.GetStateHash()
	if got != want {
		return fmt.Errorf("unexpected state root hash, got: %v, want: %v", got, want)
	}

	return ctx.State.Error()
}

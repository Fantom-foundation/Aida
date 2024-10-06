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
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeEthereumDbPostTransactionUpdator creates an extension which fixes Ethereum exceptions in LiveDB
func MakeEthereumDbPostTransactionUpdator(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	if cfg.ChainID != utils.EthereumChainID {
		return extension.NilExtension[txcontext.TxContext]{}
	}

	log := logger.NewLogger(cfg.LogLevel, "Ethereum-Exception-Updator")

	return makeEthereumDbPostTransactionUpdator(cfg, log)
}

func makeEthereumDbPostTransactionUpdator(cfg *utils.Config, log logger.Logger) executor.Extension[txcontext.TxContext] {
	return &ethereumDbPostTransactionUpdater{
		cfg: cfg,
		log: log,
	}
}

// PostTransaction fixes OutputAlloc ethereum exceptions in given substate
func (v *ethereumDbPostTransactionUpdater) PostTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	return updateEthereumDb(state, ctx.State, false)
}

type ethereumDbPostTransactionUpdater struct {
	extension.NilExtension[txcontext.TxContext]
	cfg *utils.Config
	log logger.Logger
}

// PreRun informs the user that ethereumExceptionUpdator is enabled.
func (v *ethereumDbPostTransactionUpdater) PreRun(executor.State[txcontext.TxContext], *executor.Context) error {
	v.log.Warning("Ethereum exception post transaction updator is enabled.")

	return nil
}

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

// MakeEthereumDbPreTransactionUpdater creates an extension which fixes Ethereum exceptions in pre transaction in LiveDB
func MakeEthereumDbPreTransactionUpdater(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	log := logger.NewLogger(cfg.LogLevel, "Ethereum-Exception-Updater")

	return makeEthereumDbPreTransactionUpdater(cfg, log)
}

func makeEthereumDbPreTransactionUpdater(cfg *utils.Config, log logger.Logger) executor.Extension[txcontext.TxContext] {
	if cfg.ChainID != utils.EthereumChainID {
		return extension.NilExtension[txcontext.TxContext]{}
	}

	return &ethereumDbPreTransactionUpdater{
		cfg: cfg,
		log: log,
	}
}

type ethereumDbPreTransactionUpdater struct {
	extension.NilExtension[txcontext.TxContext]
	cfg *utils.Config
	log logger.Logger
}

// PreTransaction fixes InputSubstate ethereum exceptions in given substate
func (v *ethereumDbPreTransactionUpdater) PreTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	return updateStateDbOnEthereumChain(state.Data.GetInputState(), ctx.State, false)
}

// PreRun informs the user that ethereumDbPreTransactionUpdater is enabled.
func (v *ethereumDbPreTransactionUpdater) PreRun(executor.State[txcontext.TxContext], *executor.Context) error {
	v.log.Warning("Ethereum exception pre transaction updater is enabled.")

	return nil
}

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

package primer

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeTxPrimer creates an extension that primes StateDb before each transaction
func MakeTxPrimer(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	return makeTxPrimer(cfg, logger.NewLogger(cfg.LogLevel, "TxPrimer"))
}

func makeTxPrimer(cfg *utils.Config, log logger.Logger) executor.Extension[txcontext.TxContext] {
	return &txPrimer{cfg: cfg, log: log}
}

type txPrimer struct {
	extension.NilExtension[txcontext.TxContext]
	primeCtx *utils.PrimeContext
	cfg      *utils.Config
	log      logger.Logger
}

func (p *txPrimer) PreRun(_ executor.State[txcontext.TxContext], ctx *executor.Context) error {
	p.primeCtx = utils.NewPrimeContext(p.cfg, ctx.State, p.log)
	return nil
}

// PreTransaction primes StateDb
func (p *txPrimer) PreTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	return p.primeCtx.PrimeStateDB(state.Data.GetInputState(), ctx.State)
}

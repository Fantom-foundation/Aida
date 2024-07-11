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

package executor

import (
	"fmt"

	statetest "github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

func NewEthStateTestProvider(cfg *utils.Config) Provider[txcontext.TxContext] {
	return ethTestProvider{cfg, logger.NewLogger(cfg.LogLevel, "eth-state-test-provider")}
}

type ethTestProvider struct {
	cfg *utils.Config
	log logger.Logger
}

func (e ethTestProvider) Run(_ int, _ int, consumer Consumer[txcontext.TxContext]) error {
	b, err := statetest.OpenStateTests(e.cfg.ArgPath)
	if err != nil {
		return err
	}

	// todo move file logging to eth_state_test_logger
	var (
		numTestFiles int
		maxTestFiles = len(b)
	)

	e.log.Noticef("There is %v test files.", maxTestFiles)
	defer func() {
		e.log.Noticef("Progress: %v / %v files iterated.", numTestFiles, maxTestFiles)
	}()

	// iterate all state json files
	for _, t := range b {
		numTestFiles++
		// divide them by fork
		for i, dt := range t.Divide(e.cfg.ChainID) {
			err = consumer(TransactionInfo[txcontext.TxContext]{
				Block:       int(dt.Env.GetNumber()),
				Transaction: i,
				Data:        dt,
			})
			if err != nil {
				return fmt.Errorf("transaction failed, file: %v, err: %w", dt.FilePath, err)
			}
		}
		if numTestFiles%100 == 0 {
			e.log.Noticef("Progress: %v / %v files iterated.", numTestFiles, maxTestFiles)
		}
	}

	return nil
}

func (e ethTestProvider) Close() {
	// ignored
}

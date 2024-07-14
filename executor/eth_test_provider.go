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
	b, err := statetest.NewDecoder(e.cfg)
	if err != nil {
		return err
	}

	for i, testCase := range b.DivideStateTests() {
		err = consumer(TransactionInfo[txcontext.TxContext]{
			// Blocks 0 and 1 are used by priming
			Block:       2,
			Transaction: i,
			Data:        testCase,
		})
		if err != nil {
			return fmt.Errorf("transaction failed\n%s\nerr: %w", testCase, err)
		}
	}

	return nil
}

func (e ethTestProvider) Close() {
	// ignored
}

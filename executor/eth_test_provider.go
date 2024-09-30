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
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
)

func NewEthStateTestProvider(cfg *utils.Config) (Provider[txcontext.TxContext], []common.Hash, error) {
	splitter, err := statetest.NewTestCaseSplitter(cfg)
	if err != nil {
		return nil, nil, err
	}

	tests, rootHashes, err := splitter.SplitStateTests()
	if err != nil {
		return nil, nil, err
	}

	return ethTestProvider{tests}, rootHashes, nil
}

type ethTestProvider struct {
	tests []statetest.Transaction
}

func (e ethTestProvider) Run(_ int, _ int, consumer Consumer[txcontext.TxContext]) error {
	for i, tx := range e.tests {
		err := consumer(TransactionInfo[txcontext.TxContext]{
			// Blocks 0 and 1 are used by priming
			Block:       2,
			Transaction: i,
			Data:        tx.Ctx,
		})
		if err != nil {
			return fmt.Errorf("transaction failed\n%s\nerr: %w", tx.Ctx, err)
		}
	}

	return nil
}

func (e ethTestProvider) Close() {
	// ignored
}

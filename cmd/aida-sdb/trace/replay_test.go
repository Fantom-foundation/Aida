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

package trace

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestSdbReplay_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[[]operation.Operation](ctrl)
	processor := executor.NewMockProcessor[[]operation.Operation](ctrl)
	ext := executor.NewMockExtension[[]operation.Operation](ctrl)

	cfg := &utils.Config{}
	cfg.DbImpl = "carmen"
	cfg.KeepDb = false

	cfg.First = 0
	cfg.Last = 0

	provider.EXPECT().
		Run(0, 1, gomock.Any()).
		DoAndReturn(func(from int, to int, consumer executor.Consumer[[]operation.Operation]) error {
			for i := from; i < to; i++ {
				consumer(executor.TransactionInfo[[]operation.Operation]{Block: 0, Transaction: 0, Data: testOperationsA})
				consumer(executor.TransactionInfo[[]operation.Operation]{Block: 0, Transaction: 1, Data: testOperationsB})
			}
			return nil
		})

	// All transactions are processed in order
	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[[]operation.Operation](0), gomock.Any()),

		// tx 0
		ext.EXPECT().PreTransaction(executor.AtTransaction[[]operation.Operation](0, 0), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[[]operation.Operation](0, 0), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[[]operation.Operation](0, 0), gomock.Any()),

		// tx 1
		ext.EXPECT().PreTransaction(executor.AtTransaction[[]operation.Operation](0, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[[]operation.Operation](0, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[[]operation.Operation](0, 1), gomock.Any()),

		ext.EXPECT().PostRun(executor.AtBlock[[]operation.Operation](1), gomock.Any(), nil),
	)

	if err := replay(cfg, provider, processor, []executor.Extension[[]operation.Operation]{ext}); err != nil {
		t.Errorf("record failed: %v", err)
	}
}

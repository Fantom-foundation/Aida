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
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/mock/gomock"
)

func TestEthStateTestLogHashValidator_PostTransactionChecksLogsHash(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ContinueOnFailure = false
	ext := makeEthStateTestLogHashValidator(cfg)
	ctrl := gomock.NewController(t)
	res := txcontext.NewMockResult(ctrl)
	receipt := txcontext.NewMockReceipt(ctrl)

	tests := []struct {
		name         string
		data         txcontext.TxContext
		returnedLogs []*types.Log
		wantError    string
	}{
		{
			name:         "same_hashes",
			data:         ethtest.CreateTestTransaction(t),
			returnedLogs: []*types.Log{},
		},
		{
			name: "different_hashes",
			data: ethtest.CreateErrorTestTransaction(t),
			returnedLogs: []*types.Log{
				{}, // add something to produce different hash
			},
			wantError: "unexpected logs hash",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res.EXPECT().GetReceipt().Return(receipt)
			receipt.EXPECT().GetLogs().Return(test.returnedLogs)
			ctx := &executor.Context{ExecutionResult: res}
			st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: test.data}

			err := ext.PostTransaction(st, ctx)
			if err == nil && test.wantError == "" {
				return
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Errorf("unexpected error;\ngot: %v\nwant: %v", err, test.wantError)
			}
		})
	}
}

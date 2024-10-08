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
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

func TestEthStateTestValidator_PostBlockCheckStateRoot(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ContinueOnFailure = false
	ext := makeEthStateTestStateHashValidator(cfg)

	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	tests := []struct {
		name          string
		ctx           txcontext.TxContext
		gotHash       common.Hash
		expectedError error
	}{
		{
			name:          "same_hashes",
			ctx:           ethtest.CreateTestTransactionWithHash(t, common.Hash{1}),
			gotHash:       common.Hash{1},
			expectedError: nil,
		},
		{
			name:          "different_hashes",
			ctx:           ethtest.CreateTestTransactionWithHash(t, common.Hash{1}),
			gotHash:       common.Hash{2},
			expectedError: fmt.Errorf("unexpected root hash, got: %s, want: %s", common.Hash{2}, common.Hash{1}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db.EXPECT().GetHash().Return(test.gotHash, nil)
			ctx := &executor.Context{State: db}
			st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: test.ctx}

			err := ext.PostBlock(st, ctx)
			if err == nil && test.expectedError == nil {
				return
			}
			if got, want := err, test.expectedError; !strings.EqualFold(got.Error(), want.Error()) {
				t.Errorf("unexpected error, got: %v, want: %v", got, want)
			}
		})
	}
}

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
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
)

func TestEthStateTestErrorValidator_PreTransactionReturnsError(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ContinueOnFailure = true

	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	data := ethtest.CreateTestTransaction(t)
	ctx := new(executor.Context)
	ctx.State = db
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: data}

	gomock.InOrder(
		db.EXPECT().Exist(common.HexToAddress("0x1")).Return(true),
		db.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1)),
		db.EXPECT().GetNonce(common.HexToAddress("0x1")),
		db.EXPECT().GetCode(common.HexToAddress("0x1")),
	)

	gomock.InOrder(
		db.EXPECT().Exist(common.HexToAddress("0x2")).Return(true),
		db.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1)),
		db.EXPECT().GetNonce(common.HexToAddress("0x2")),
		db.EXPECT().GetCode(common.HexToAddress("0x2")),
	)

	ext := makeEthStateTestErrorValidator(cfg, log)
	err := ext.PreTransaction(st, ctx)
	if err == nil {
		t.Fatal("pre-transaction must return error")
	}
}

func TestEthStateTestErrorValidator_PostTransactionCheckError(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ContinueOnFailure = false
	ext := makeEthStateTestErrorValidator(cfg, nil)

	tests := []struct {
		name      string
		expect    txcontext.Result
		get       txcontext.TxContext
		wantError error
	}{
		{
			name:      "expectNoErrorGetNoError",
			expect:    ethtest.CreateNoErrorTestTransaction(t).GetResult(),
			get:       ethtest.CreateNoErrorTestTransaction(t),
			wantError: nil,
		},
		{
			name:      "expectErrorGetError",
			expect:    ethtest.CreateErrorTestTransaction(t).GetResult(),
			get:       ethtest.CreateErrorTestTransaction(t),
			wantError: nil,
		},
		{
			name:      "expectErrorGetNoError",
			expect:    ethtest.CreateNoErrorTestTransaction(t).GetResult(),
			get:       ethtest.CreateErrorTestTransaction(t),
			wantError: fmt.Errorf("expected error %w, got no error", errors.New("err")),
		},
		{
			name:      "expectNoErrorGetError",
			expect:    ethtest.CreateErrorTestTransaction(t).GetResult(),
			get:       ethtest.CreateNoErrorTestTransaction(t),
			wantError: fmt.Errorf("unexpected error: %w", errors.New("err")),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := &executor.Context{ExecutionResult: test.expect}
			st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: test.get}

			err := ext.PostTransaction(st, ctx)
			if err == nil && test.wantError == nil {
				return
			}
			if !strings.Contains(err.Error(), test.wantError.Error()) {
				t.Errorf("unexpected error;\ngot: %v\nwant: %v", err, test.wantError)
			}
		})
	}
}

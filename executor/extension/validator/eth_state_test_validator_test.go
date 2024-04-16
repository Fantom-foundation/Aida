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
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

func TestEthStatePrepper_PostTransactionLogsErrorAndDoesNotReturnIt(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ContinueOnFailure = true

	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	data := ethtest.CreateTestData(t)
	ctx := new(executor.Context)
	ctx.State = db
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: data}

	got := common.HexToHash("0x01")
	want := data.GetStateHash()

	expectedErr := fmt.Errorf("%v - (%v) FAIL\ndifferent hashes\ngot: %v\nwant:%v", "TestLabel", "TestNetwork", got.Hex(), want.Hex())

	gomock.InOrder(
		db.EXPECT().GetHash().Return(got, nil),
		log.EXPECT().Error(expectedErr),
	)

	ext := makeEthStateTestValidator(cfg, log)

	err := ext.PostTransaction(st, ctx)
	if err != nil {
		t.Fatalf("post-transaction cannot return error; %v", err)
	}
}

func TestEthStatePrepper_PostTransactionLogsPass(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ContinueOnFailure = false

	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	data := ethtest.CreateTestData(t)
	ctx := new(executor.Context)
	ctx.State = db
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: data}

	want := data.GetStateHash()

	gomock.InOrder(
		db.EXPECT().GetHash().Return(want, nil),
		log.EXPECT().Noticef("%v - (%v) PASS\nblock: %v; tx: %v\nhash:%v", "TestLabel", "TestNetwork", 1, 1, want.Hex()),
	)

	ext := makeEthStateTestValidator(cfg, log)

	err := ext.PostTransaction(st, ctx)
	if err != nil {
		t.Fatalf("post-transaction cannot return error; %v", err)
	}
}

func TestEthStatePrepper_PostTransactionReturnsError(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ContinueOnFailure = false

	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	data := ethtest.CreateTestData(t)
	ctx := new(executor.Context)
	ctx.State = db
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: data}

	got := common.HexToHash("0x01")
	want := data.GetStateHash()

	db.EXPECT().GetHash().Return(got, nil)

	ext := makeEthStateTestValidator(cfg, log)

	err := ext.PostTransaction(st, ctx)
	if err == nil {
		t.Fatalf("post-transaction must return error; %v", err)
	}

	expectedErr := fmt.Errorf("%v - (%v) FAIL\ndifferent hashes\ngot: %v\nwant:%v", "TestLabel", "TestNetwork", got.Hex(), want.Hex())
	if strings.Compare(err.Error(), expectedErr.Error()) != 0 {
		t.Fatalf("unexpected error\ngot:%v\nwant:%v", err.Error(), expectedErr.Error())
	}
}

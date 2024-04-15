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

package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

func TestStatePrepper_PreparesStateBeforeEachTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	allocA := substatecontext.NewTxContext(&substate.Substate{InputAlloc: substate.SubstateAlloc{common.Address{1}: &substate.SubstateAccount{}}})
	allocB := substatecontext.NewTxContext(&substate.Substate{InputAlloc: substate.SubstateAlloc{common.Address{2}: &substate.SubstateAccount{}}})
	ctx := &executor.Context{State: db}

	gomock.InOrder(
		db.EXPECT().PrepareSubstate(allocA.GetInputState(), uint64(5)),
		db.EXPECT().PrepareSubstate(allocB.GetInputState(), uint64(7)),
	)

	prepper := MakeStateDbPrepper()

	prepper.PreTransaction(executor.State[txcontext.TxContext]{
		Block: 5,
		Data:  allocA,
	}, ctx)

	prepper.PreTransaction(executor.State[txcontext.TxContext]{
		Block: 7,
		Data:  allocB,
	}, ctx)
}

func TestStatePrepper_DoesNotCrashOnMissingStateOrSubstate(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}

	prepper := MakeStateDbPrepper()
	prepper.PreTransaction(executor.State[txcontext.TxContext]{Block: 5}, nil)                                                           // misses both
	prepper.PreTransaction(executor.State[txcontext.TxContext]{Block: 5}, ctx)                                                           // misses the data
	prepper.PreTransaction(executor.State[txcontext.TxContext]{Block: 5, Data: substatecontext.NewTxContext(&substate.Substate{})}, nil) // misses the state
}

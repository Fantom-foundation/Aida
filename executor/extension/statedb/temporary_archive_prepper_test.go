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

package statedb

import (
	"encoding/json"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/state"
	"go.uber.org/mock/gomock"
)

func TestTemporaryArchivePrepper_PreTransactionGetsArchiveForRequestedBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ext := MakeTemporaryArchivePrepper()

	db.EXPECT().GetArchiveState(uint64(10)).Return(nil, nil)

	st := executor.State[*rpc.RequestAndResults]{Block: 10, Transaction: 0, Data: data}
	ctx := &executor.Context{State: db}
	err := ext.PreTransaction(st, ctx)
	if err != nil {
		t.Fatal(err)
	}
}

var data = &rpc.RequestAndResults{
	RequestedBlock: 10,
	Query: &rpc.Body{
		Params: []interface{}{"test", "pending"},
	},
	Response: &rpc.Response{
		Version:   "2.0",
		ID:        json.RawMessage{1},
		BlockID:   10,
		Timestamp: 10,
	},
	SkipValidation: false,
}

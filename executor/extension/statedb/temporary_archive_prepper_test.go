package statedb

import (
	"encoding/json"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/state"
	"go.uber.org/mock/gomock"
)

func TestTemporaryArchivePrepper_PreTransactionMarksPendingBlockAsSkipValidation(t *testing.T) {
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

	if !data.SkipValidation {
		t.Fatal("SkipValidation must be true")
	}
}

var data = &rpc.RequestAndResults{
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

package validator

import (
	"errors"
	"strings"
	"testing"

	statetest "github.com/Fantom-foundation/Aida/ethtest/state_test"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestShadowDbValidator_PostBlockReturnsNilError(t *testing.T) {
	cfg := &utils.Config{}

	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	data := statetest.CreateTestData(t)
	ctx := new(executor.Context)
	ctx.State = db
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: data}

	gomock.InOrder(
		db.EXPECT().GetHash(),
		db.EXPECT().Error().Return(nil),
	)

	ext := makeShadowDbValidator(cfg)

	err := ext.PostBlock(st, ctx)
	if err != nil {
		t.Fatalf("post-transaction cannot return error; %v", err)
	}
}

func TestShadowDbValidator_PostBlockReturnsError(t *testing.T) {
	cfg := &utils.Config{}

	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	data := statetest.CreateTestData(t)
	ctx := new(executor.Context)
	ctx.State = db
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: data}

	expectedErr := errors.New("FAIL")

	gomock.InOrder(
		db.EXPECT().GetHash(),
		db.EXPECT().Error().Return(expectedErr),
	)

	ext := makeShadowDbValidator(cfg)

	err := ext.PostBlock(st, ctx)
	if err == nil {
		t.Fatalf("post-transaction must return error; %v", err)
	}

	if strings.Compare(err.Error(), expectedErr.Error()) != 0 {
		t.Fatalf("unexpected error\ngot:%v\nwant:%v", err.Error(), expectedErr.Error())
	}
}

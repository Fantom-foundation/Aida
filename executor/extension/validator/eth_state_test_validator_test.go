package validator

import (
	"fmt"
	"strings"
	"testing"

	statetest "github.com/Fantom-foundation/Aida/ethtest/statetest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

func TestEthStateTestValidator_PostTransactionLogsErrorAndDoesNotReturnIt(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ContinueOnFailure = true

	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	data := statetest.CreateTestData(t)
	ctx := new(executor.Context)
	ctx.State = db
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: data}

	got := common.HexToHash("0x01")
	want := data.GetStateHash()

	expectedErr := fmt.Errorf("%v - (%v) FAIL\ndifferent hashes\ngot: %v\nwant:%v", "TestLabel", "TestNetwork", got.Hex(), want.Hex())

	gomock.InOrder(
		db.EXPECT().GetHash().Return(got),
		log.EXPECT().Error(expectedErr),
	)

	ext := makeEthStateTestValidator(cfg, log)

	err := ext.PostBlock(st, ctx)
	if err != nil {
		t.Fatalf("post-transaction cannot return error; %v", err)
	}
}

func TestEthStateTestValidator_PostTransactionLogsPass(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ContinueOnFailure = false

	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	data := statetest.CreateTestData(t)
	ctx := new(executor.Context)
	ctx.State = db
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: data}

	want := data.GetStateHash()

	gomock.InOrder(
		db.EXPECT().GetHash().Return(want),
		log.EXPECT().Noticef("%v - (%v) PASS\nblock: %v; tx: %v\nhash:%v", "TestLabel", "TestNetwork", 1, 1, want.Hex()),
	)

	ext := makeEthStateTestValidator(cfg, log)

	err := ext.PostBlock(st, ctx)
	if err != nil {
		t.Fatalf("post-transaction cannot return error; %v", err)
	}
}

func TestEthStateTestValidator_PostTransactionReturnsError(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ContinueOnFailure = false

	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	data := statetest.CreateTestData(t)
	ctx := new(executor.Context)
	ctx.State = db
	st := executor.State[txcontext.TxContext]{Block: 1, Transaction: 1, Data: data}

	got := common.HexToHash("0x01")
	want := data.GetStateHash()

	db.EXPECT().GetHash().Return(got)

	ext := makeEthStateTestValidator(cfg, log)

	err := ext.PostBlock(st, ctx)
	if err == nil {
		t.Fatalf("post-transaction must return error; %v", err)
	}

	expectedErr := fmt.Errorf("%v - (%v) FAIL\ndifferent hashes\ngot: %v\nwant:%v", "TestLabel", "TestNetwork", got.Hex(), want.Hex())
	if strings.Compare(err.Error(), expectedErr.Error()) != 0 {
		t.Fatalf("unexpected error\ngot:%v\nwant:%v", err.Error(), expectedErr.Error())
	}
}

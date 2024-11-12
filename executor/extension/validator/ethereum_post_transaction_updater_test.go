package validator

import (
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
)

func TestEthereumPostTransactionUpdator_Skips(t *testing.T) {
	tests := []struct {
		name   string
		vmImpl string
		block  int
	}{
		{
			name:   "SkipNonLfvm",
			vmImpl: "geth",
			block:  getExceptionBlock(),
		},
		{
			name:   "SkipExceptionOnWorkingBlock",
			vmImpl: "lfvm",
			block:  10000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &utils.Config{}
			cfg.VmImpl = tt.vmImpl

			ctrl := gomock.NewController(t)
			log := logger.NewMockLogger(ctrl)
			db := state.NewMockStateDB(ctrl)

			data := ethtest.CreateTestTransaction(t)
			ctx := new(executor.Context)
			ctx.State = db

			st := executor.State[txcontext.TxContext]{Block: tt.block, Transaction: 1, Data: data}

			ext := makeEthereumDbPostTransactionUpdater(cfg, log)
			err := ext.PostTransaction(st, ctx)
			if err != nil {
				t.Fatal("post-transaction unexpected error: ", err)
			}
		})
	}
}

func TestEthereumPostTransactionUpdator_OverwriteAccount(t *testing.T) {
	cfg := &utils.Config{}
	cfg.VmImpl = "lfvm"

	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	data := ethtest.CreateTestTransaction(t)
	ctx := new(executor.Context)
	ctx.State = db
	st := executor.State[txcontext.TxContext]{Block: getExceptionBlock(), Transaction: 1, Data: data}

	gomock.InOrder(
		db.EXPECT().Exist(common.HexToAddress("0x1")).Return(true),
		db.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1)),
		db.EXPECT().SubBalance(common.HexToAddress("0x1"), uint256.NewInt(1), tracing.BalanceChangeUnspecified),
		db.EXPECT().AddBalance(common.HexToAddress("0x1"), uint256.NewInt(1000), tracing.BalanceChangeUnspecified),
		db.EXPECT().GetNonce(common.HexToAddress("0x1")),
		db.EXPECT().SetNonce(common.HexToAddress("0x1"), gomock.Any()),
		db.EXPECT().GetCode(common.HexToAddress("0x1")),
		db.EXPECT().GetState(common.HexToAddress("0x1"), common.HexToHash("0x1")),
		db.EXPECT().SetState(common.HexToAddress("0x1"), common.HexToHash("0x1"), common.HexToHash("0x2")),
	)

	gomock.InOrder(
		db.EXPECT().Exist(common.HexToAddress("0x2")).Return(true),
		db.EXPECT().GetBalance(common.HexToAddress("0x2")).Return(uint256.NewInt(2)),
		db.EXPECT().SubBalance(common.HexToAddress("0x2"), uint256.NewInt(2), tracing.BalanceChangeUnspecified),
		db.EXPECT().AddBalance(common.HexToAddress("0x2"), uint256.NewInt(2000), tracing.BalanceChangeUnspecified),
		db.EXPECT().GetNonce(common.HexToAddress("0x2")),
		db.EXPECT().SetNonce(common.HexToAddress("0x2"), gomock.Any()),
		db.EXPECT().GetCode(common.HexToAddress("0x2")).Return([]byte{0x1}),
		db.EXPECT().SetCode(common.HexToAddress("0x2"), gomock.Any()),
	)

	ext := makeEthereumDbPostTransactionUpdater(cfg, log)
	err := ext.PostTransaction(st, ctx)
	if err != nil {
		t.Fatal("post-transaction unexpected error: ", err)
	}
}

func getExceptionBlock() int {
	// retrieving exception block
	var exceptionBlock int
	for key := range ethereumLfvmBlockExceptions {
		exceptionBlock = key
		break
	}
	return exceptionBlock
}

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
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
)

func TestEthereumPreTransactionUpdator_FixBalance(t *testing.T) {
	cfg := &utils.Config{}

	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	data := ethtest.CreateTestTransaction(t)
	ctx := new(executor.Context)
	ctx.State = db
	st := executor.State[txcontext.TxContext]{Block: getExceptionBlock(), Transaction: 1, Data: data}

	gomock.InOrder(
		db.EXPECT().Exist(common.HexToAddress("0x1")).Return(true),
		db.EXPECT().GetBalance(common.HexToAddress("0x1")).Return(uint256.NewInt(1)),
		db.EXPECT().SubBalance(common.HexToAddress("0x1"), uint256.NewInt(1), tracing.BalanceChangeUnspecified),
		db.EXPECT().AddBalance(common.HexToAddress("0x1"), uint256.NewInt(1000), tracing.BalanceChangeUnspecified),
	)

	gomock.InOrder(
		db.EXPECT().Exist(common.HexToAddress("0x2")).Return(true),
		db.EXPECT().GetBalance(common.HexToAddress("0x2")).Return(uint256.NewInt(2000)),
	)

	ext := makeEthereumDbPreTransactionUpdater(cfg, log)
	err := ext.PreTransaction(st, ctx)
	if err != nil {
		t.Fatal("post-transaction unexpected error: ", err)
	}
}

func TestEthereumPreTransactionUpdator_DontFixBalanceIfLower(t *testing.T) {
	cfg := &utils.Config{}

	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	data := ethtest.CreateTestTransaction(t)
	ctx := new(executor.Context)
	ctx.State = db
	st := executor.State[txcontext.TxContext]{Block: getExceptionBlock(), Transaction: 1, Data: data}

	gomock.InOrder(
		db.EXPECT().Exist(common.HexToAddress("0x1")).Return(true),
		db.EXPECT().GetBalance(common.HexToAddress("0x1")).Return(uint256.NewInt(10000)),
	)

	gomock.InOrder(
		db.EXPECT().Exist(common.HexToAddress("0x2")).Return(true),
		db.EXPECT().GetBalance(common.HexToAddress("0x2")).Return(uint256.NewInt(2000)),
	)

	ext := makeEthereumDbPreTransactionUpdater(cfg, log)
	err := ext.PreTransaction(st, ctx)
	if err != nil {
		t.Fatal("post-transaction unexpected error: ", err)
	}
}

func TestEthereumPreTransactionUpdator_BeaconRootsAddressStorageException(t *testing.T) {
	cfg := &utils.Config{}

	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	data := ethtest.CreateBeaconRootsAddressTestTransaction(t)

	ctx := new(executor.Context)
	ctx.State = db
	st := executor.State[txcontext.TxContext]{Block: getExceptionBlock(), Transaction: 1, Data: data}

	gomock.InOrder(
		db.EXPECT().Exist(params.BeaconRootsAddress).Return(true),
		db.EXPECT().GetBalance(params.BeaconRootsAddress).Return(uint256.NewInt(1)),
		db.EXPECT().GetState(params.BeaconRootsAddress, common.HexToHash("0x1")),
		db.EXPECT().SetState(params.BeaconRootsAddress, common.HexToHash("0x1"), common.HexToHash("0x2")),
	)

	ext := makeEthereumDbPreTransactionUpdater(cfg, log)
	err := ext.PreTransaction(st, ctx)
	if err != nil {
		t.Fatal("post-transaction unexpected error: ", err)
	}
}

func TestEthereumPreTransactionUpdator_DaoFork(t *testing.T) {
	cfg := &utils.Config{}

	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	data := ethtest.CreateDaoForkAddressTestTransaction(t)

	ctx := new(executor.Context)
	ctx.State = db
	st := executor.State[txcontext.TxContext]{Block: getExceptionBlock(), Transaction: 1, Data: data}

	gomock.InOrder(
		db.EXPECT().Exist(params.DAODrainList()[0]).Return(true),
		db.EXPECT().GetBalance(params.DAODrainList()[0]).Return(uint256.NewInt(1)),
		db.EXPECT().SubBalance(params.DAODrainList()[0], uint256.NewInt(1), tracing.BalanceChangeUnspecified),
		db.EXPECT().AddBalance(params.DAODrainList()[0], uint256.NewInt(0), tracing.BalanceChangeUnspecified),
	)

	ext := makeEthereumDbPreTransactionUpdater(cfg, log)
	err := ext.PreTransaction(st, ctx)
	if err != nil {
		t.Fatal("post-transaction unexpected error: ", err)
	}
}

package executor

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/params"
	"go.uber.org/mock/gomock"
)

func TestNormaTxProvider_Run(t *testing.T) {
	ctrl := gomock.NewController(t)
	dbMock := state.NewMockStateDB(ctrl)

	cfg := &utils.Config{
		BlockLength:     uint64(3),
		TxGeneratorType: []string{"counter"},
		ChainID:         297,
	}
	provider := NewNormaTxProvider(cfg, dbMock)
	consumer := NewMockTxConsumer(ctrl)

	gomock.InOrder(
		// treasure account initialization
		dbMock.EXPECT().BeginBlock(gomock.Any()),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().CreateAccount(gomock.Any()),
		dbMock.EXPECT().AddBalance(gomock.Any(), gomock.Any()),
		dbMock.EXPECT().EndTransaction(),
		dbMock.EXPECT().EndBlock(),

		// contract deployment

		// expected on block 2, because block 1 is treasure account initialization
		// and we are starting from block 1
		consumer.EXPECT().Consume(2, 0, gomock.Any()).Return(nil),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),

		// funding accounts
		// we return a lot of tokens, so we don't have to "fund" them
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(0).Mul(big.NewInt(params.Ether), big.NewInt(1_000_000))),
		// nonce for account deploying the contract has to be 1
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(1)),
		// nonce for funded accounts has to be 0
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(0).Mul(big.NewInt(params.Ether), big.NewInt(1_000_000))),

		// waiting for contract deployment requires checking the nonce
		// of funded accounts, since we did no funding, then nonce is 0
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)).AnyTimes(),

		// generating transactions, starting from transaction 1 (0 was contract deployment)
		consumer.EXPECT().Consume(2, 1, gomock.Any()).Return(nil),
		consumer.EXPECT().Consume(2, 2, gomock.Any()).Return(nil),
		consumer.EXPECT().Consume(3, 0, gomock.Any()).Return(nil),
		consumer.EXPECT().Consume(3, 1, gomock.Any()).Return(nil),
		consumer.EXPECT().Consume(3, 2, gomock.Any()).Return(nil),
	)

	err := provider.Run(1, 3, toSubstateConsumer(consumer))
	if err != nil {
		t.Fatalf("failed to run provider: %v", err)
	}
}

func TestNormaTxProvider_RunAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	dbMock := state.NewMockStateDB(ctrl)

	cfg := &utils.Config{
		BlockLength:     uint64(5),
		TxGeneratorType: []string{"erc20", "counter", "store"},
		ChainID:         297,
	}
	provider := NewNormaTxProvider(cfg, dbMock)
	consumer := NewMockTxConsumer(ctrl)

	balance := big.NewInt(0).Mul(big.NewInt(params.Ether), big.NewInt(1_000_000))

	gomock.InOrder(
		// treasure account initialization
		dbMock.EXPECT().BeginBlock(gomock.Any()),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().CreateAccount(gomock.Any()),
		dbMock.EXPECT().AddBalance(gomock.Any(), gomock.Any()),
		dbMock.EXPECT().EndTransaction(),
		dbMock.EXPECT().EndBlock(),

		// contract deployment in order: erc20 -> counter -> store

		// expected on block 2, because block 1 is treasure account initialization
		// and we are starting from block 1

		// ERC 20
		consumer.EXPECT().Consume(2, 0, gomock.Any()).Return(nil),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(balance),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(1)),
		// funding accounts
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(balance),
		// mint nf tokens
		consumer.EXPECT().Consume(2, 1, gomock.Any()).Return(nil),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(1)),

		// COUNTER
		consumer.EXPECT().Consume(2, 2, gomock.Any()).Return(nil),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(1)),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(balance),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(2)),
		// funding accounts
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(balance),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(1)),

		// STORE
		consumer.EXPECT().Consume(2, 3, gomock.Any()).Return(nil),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(2)),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(balance),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(3)),
		// funding accounts
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(balance),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(2)),

		// generating transactions
		consumer.EXPECT().Consume(2, 4, gomock.Any()).Return(nil),
		consumer.EXPECT().Consume(3, 0, gomock.Any()).Return(nil),
		consumer.EXPECT().Consume(3, 1, gomock.Any()).Return(nil),
		consumer.EXPECT().Consume(3, 2, gomock.Any()).Return(nil),
		consumer.EXPECT().Consume(3, 3, gomock.Any()).Return(nil),
		consumer.EXPECT().Consume(3, 4, gomock.Any()).Return(nil),
	)

	err := provider.Run(1, 3, toSubstateConsumer(consumer))
	if err != nil {
		t.Fatalf("failed to run provider: %v", err)
	}
}

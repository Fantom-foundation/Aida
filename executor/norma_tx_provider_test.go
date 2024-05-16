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
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),
		dbMock.EXPECT().EndTransaction(),

		// funding accounts
		// we return a lot of tokens, so we don't have to "fund" them
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(0).Mul(big.NewInt(params.Ether), big.NewInt(1_000_000))),
		dbMock.EXPECT().EndTransaction(),
		// nonce for account deploying the contract has to be 1
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(1)),
		dbMock.EXPECT().EndTransaction(),
		// nonce for funded accounts has to be 0
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),
		dbMock.EXPECT().EndTransaction(),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(0).Mul(big.NewInt(params.Ether), big.NewInt(1_000_000))),
		dbMock.EXPECT().EndTransaction(),

		// waiting for contract deployment requires checking the nonce
		// of funded accounts, since we did no funding, then nonce is 0
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),
		dbMock.EXPECT().EndTransaction(),

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
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),
		dbMock.EXPECT().EndTransaction(),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(balance),
		dbMock.EXPECT().EndTransaction(),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(1)),
		dbMock.EXPECT().EndTransaction(),
		// funding accounts
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),
		dbMock.EXPECT().EndTransaction(),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(balance),
		dbMock.EXPECT().EndTransaction(),
		// mint nf tokens
		consumer.EXPECT().Consume(2, 1, gomock.Any()).Return(nil),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(1)),
		dbMock.EXPECT().EndTransaction(),
		// COUNTER
		consumer.EXPECT().Consume(2, 2, gomock.Any()).Return(nil),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(1)),
		dbMock.EXPECT().EndTransaction(),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(balance),
		dbMock.EXPECT().EndTransaction(),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(2)),
		dbMock.EXPECT().EndTransaction(),
		// funding accounts
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),
		dbMock.EXPECT().EndTransaction(),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(balance),
		dbMock.EXPECT().EndTransaction(),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(1)),
		dbMock.EXPECT().EndTransaction(),
		// STORE
		consumer.EXPECT().Consume(2, 3, gomock.Any()).Return(nil),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(2)),
		dbMock.EXPECT().EndTransaction(),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(balance),
		dbMock.EXPECT().EndTransaction(),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(3)),
		dbMock.EXPECT().EndTransaction(),
		// funding accounts
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)),
		dbMock.EXPECT().EndTransaction(),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetBalance(gomock.Any()).Return(balance),
		dbMock.EXPECT().EndTransaction(),
		dbMock.EXPECT().BeginTransaction(gomock.Any()),
		dbMock.EXPECT().GetNonce(gomock.Any()).Return(uint64(2)),
		dbMock.EXPECT().EndTransaction(),
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

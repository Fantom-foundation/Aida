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

package ethtest

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

func CreateTestData(*testing.T) txcontext.TxContext {
	bInt := new(big.Int).SetUint64(1)
	to := common.HexToAddress("0x10")
	return &stateTestContext{
		env: &stBlockEnvironment{
			blockNumber: 1,
			Coinbase:    common.Address{},
			Difficulty:  &BigInt{*bInt},
			GasLimit:    &BigInt{*bInt},
			Number:      &BigInt{*bInt},
			Timestamp:   &BigInt{*bInt},
			BaseFee:     &BigInt{*bInt},
		},
		inputState: types.GenesisAlloc{
			common.HexToAddress("0x1"): core.GenesisAccount{
				Balance: big.NewInt(1000),
				Nonce:   1,
			},
			common.HexToAddress("0x2"): core.GenesisAccount{
				Balance: big.NewInt(2000),
				Nonce:   2,
			},
		},
		msg: &core.Message{
			To:            &to,
			From:          common.HexToAddress("0x2"),
			Nonce:         1,
			Value:         big.NewInt(1),
			GasLimit:      1,
			GasPrice:      big.NewInt(1),
			GasFeeCap:     big.NewInt(1),
			GasTipCap:     big.NewInt(1),
			Data:          []byte{0x1},
			AccessList:    make(types.AccessList, 0),
			BlobGasFeeCap: big.NewInt(1),
			BlobHashes:    make([]common.Hash, 0),
		},
	}
}

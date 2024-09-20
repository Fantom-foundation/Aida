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
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/tests"
)

var (
	data1 = hex.EncodeToString([]byte{0x1})
	data2 = hex.EncodeToString([]byte{0x2})
	data3 = hex.EncodeToString([]byte{0x3})
	data4 = hex.EncodeToString([]byte{0x4})
)

func CreateTestTransaction(t *testing.T) txcontext.TxContext {
	chainCfg, _, err := tests.GetChainConfig("Cancun")
	if err != nil {
		t.Fatalf("cannot get chain config: %v", err)
	}
	to := common.HexToAddress("0x10")
	return &stateTestContext{
		env: &stBlockEnvironment{
			blockNumber: 1,
			Coinbase:    common.Address{},
			Difficulty:  newBigInt(1),
			GasLimit:    newBigInt(1),
			Number:      newBigInt(1),
			Timestamp:   newBigInt(1),
			BaseFee:     newBigInt(1),
			chainCfg:    chainCfg,
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

func CreateTestStJson(*testing.T) *stJSON {
	return &stJSON{
		path: "test/path",
		Env: stBlockEnvironment{
			blockNumber: 1,
			Coinbase:    common.Address{0x1},
			Difficulty:  newBigInt(1),
			GasLimit:    newBigInt(1),
			Number:      newBigInt(1),
			Timestamp:   newBigInt(1),
			BaseFee:     newBigInt(1),
		},
		Pre: types.GenesisAlloc{common.Address{0x2}: types.Account{
			Code:       []byte{1},
			Storage:    make(map[common.Hash]common.Hash),
			Balance:    big.NewInt(1),
			Nonce:      1,
			PrivateKey: []byte{2},
		}},
		Tx: stTransaction{
			Data:          []string{data1, data2, data3, data4},
			GasLimit:      []*BigInt{newBigInt(1), newBigInt(2), newBigInt(3), newBigInt(4)},
			Value:         []string{data1, data2, data3, data4},
			Nonce:         newBigInt(1),
			GasPrice:      newBigInt(1),
			BlobGasFeeCap: newBigInt(1),
		},
		Out: nil,
		Post: map[string][]stPost{
			"Cancun": {
				{
					Indexes: Index{
						Data:  0,
						Gas:   0,
						Value: 0,
					},
				},
				{
					Indexes: Index{
						Data:  1,
						Gas:   1,
						Value: 1,
					},
				},
			},
			"London": {
				{
					Indexes: Index{
						Data:  2,
						Gas:   2,
						Value: 2,
					},
				},
				{
					Indexes: Index{
						Data:  3,
						Gas:   3,
						Value: 3,
					},
				},
			},
		},
	}
}

func CreateErrorTestTransaction(*testing.T) txcontext.TxContext {
	return &stateTestContext{
		expectedError: "err",
	}
}

func CreateNoErrorTestTransaction(*testing.T) txcontext.TxContext {
	return &stateTestContext{
		expectedError: "",
	}
}

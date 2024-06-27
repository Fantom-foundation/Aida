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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
)

func CreateTestData(t *testing.T) *StJSON {
	bInt := new(big.Int).SetUint64(1)
	return &StJSON{
		TestLabel: "TestLabel",
		Fork:      "TestNetwork",
		Env: stEnv{
			blockNumber: 1,
			Coinbase:    common.Address{},
			Difficulty:  &BigInt{*bInt},
			GasLimit:    &BigInt{*bInt},
			Number:      &BigInt{*bInt},
			Timestamp:   &BigInt{*bInt},
			BaseFee:     &BigInt{*bInt},
		},
		Pre: core.GenesisAlloc{
			common.HexToAddress("0x1"): core.GenesisAccount{
				Balance: big.NewInt(1000),
				Nonce:   1,
			},
			common.HexToAddress("0x2"): core.GenesisAccount{
				Balance: big.NewInt(2000),
				Nonce:   2,
			},
		},
		Tx: stTransaction{
			GasPrice:             &BigInt{*bInt},
			MaxFeePerGas:         &BigInt{*bInt},
			MaxPriorityFeePerGas: &BigInt{*bInt},
			Nonce:                &BigInt{*bInt},
			To:                   common.HexToAddress("0x10").Hex(),
			Data:                 []string{"0x"},
			GasLimit:             []*BigInt{{*bInt}},
			Value:                []string{"0x01"},
			PrivateKey:           hexutil.MustDecode("0x45a915e4d060149eb4365960e6a7a45f334393093061116b197e3240065ff2d8"),
		},
		Post: map[string][]stPostState{
			"TestNetwork": {
				{
					RootHash: common.HexToHash("0x20"),
					LogsHash: common.HexToHash("0x30"),
					indexes:  Index{},
				},
			},
		},
		//Out: hexutil.Bytes("0x0"),
	}
}

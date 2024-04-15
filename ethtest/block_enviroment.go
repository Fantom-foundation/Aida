// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package ethtest

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type stEnv struct {
	blockNumber uint64
	Coinbase    common.Address `json:"currentCoinbase"   gencodec:"required"`
	Difficulty  *BigInt        `json:"currentDifficulty" gencodec:"required"`
	GasLimit    *BigInt        `json:"currentGasLimit"   gencodec:"required"`
	Number      *BigInt        `json:"currentNumber"     gencodec:"required"`
	Timestamp   *BigInt        `json:"currentTimestamp"  gencodec:"required"`
	BaseFee     *BigInt        `json:"currentBaseFee"  gencodec:"optional"`
}

func (s *stEnv) GetCoinbase() common.Address {
	return s.Coinbase
}

func (s *stEnv) GetDifficulty() *big.Int {
	return s.Difficulty.Convert()
}

func (s *stEnv) GetGasLimit() uint64 {
	return s.GasLimit.Uint64()
}

func (s *stEnv) GetNumber() uint64 {
	return s.blockNumber
}

func (s *stEnv) GetTimestamp() uint64 {
	return s.Timestamp.Uint64()
}

func (s *stEnv) GetBlockHash(blockNumber uint64) (common.Hash, error) {
	return common.Hash{}, nil
}

func (s *stEnv) GetBaseFee() *big.Int {
	return s.BaseFee.Convert()
}

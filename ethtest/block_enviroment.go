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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

type stBlockEnvironment struct {
	blockNumber   uint64
	Coinbase      common.Address `json:"currentCoinbase"   gencodec:"required"`
	Random        *BigInt        `json:"currentRandom"        gencodec:"optional"`
	Difficulty    *BigInt        `json:"currentDifficulty" gencodec:"optional"`
	GasLimit      *BigInt        `json:"currentGasLimit"   gencodec:"required"`
	Number        *BigInt        `json:"currentNumber"     gencodec:"required"`
	Timestamp     *BigInt        `json:"currentTimestamp"  gencodec:"required"`
	BaseFee       *BigInt        `json:"currentBaseFee"  gencodec:"optional"`
	ExcessBlobGas *BigInt        `json:"currentExcessBlobGas" gencodec:"optional"`
	chainCfg      *params.ChainConfig
	fork          string
}

func (s *stBlockEnvironment) GetCoinbase() common.Address {
	return s.Coinbase
}

func (s *stBlockEnvironment) GetBlobBaseFee() *big.Int {
	if s.chainCfg.IsCancun(new(big.Int), s.Timestamp.Uint64()) && s.ExcessBlobGas != nil {
		return eip4844.CalcBlobFee(s.ExcessBlobGas.Uint64())
	}

	return nil
}

func (s *stBlockEnvironment) GetDifficulty() *big.Int {
	var difficulty *big.Int
	if s.Difficulty != nil {
		difficulty = s.Difficulty.Convert()
	}

	if s.chainCfg.IsLondon(new(big.Int)) && s.Random != nil {
		difficulty = big.NewInt(0)
	}

	return difficulty
}

func (s *stBlockEnvironment) GetGasLimit() uint64 {
	limit := s.GasLimit.Uint64()
	if limit == 0 {
		return params.GenesisGasLimit
	}

	return limit
}

func (s *stBlockEnvironment) GetNumber() uint64 {
	return s.blockNumber
}

func (s *stBlockEnvironment) GetTimestamp() uint64 {
	return s.Timestamp.Uint64()
}

func (s *stBlockEnvironment) GetBlockHash(blockNum uint64) (common.Hash, error) {
	return common.BytesToHash(crypto.Keccak256([]byte(big.NewInt(int64(blockNum)).String()))), nil
}

func (s *stBlockEnvironment) GetBaseFee() *big.Int {
	var baseFee *big.Int
	if s.chainCfg.IsLondon(new(big.Int)) {
		baseFee = s.BaseFee.Convert()
		if s.BaseFee == nil {
			// Retesteth uses `0x10` for genesis baseFee. Therefore, it defaults to
			// parent - 2 : 0xa as the basefee for 'this' context.
			baseFee = big.NewInt(0x0a)
		} else {
			baseFee = s.BaseFee.Convert()
		}
	}
	return baseFee
}

func (s *stBlockEnvironment) GetRandom() *common.Hash {
	if s.chainCfg.IsLondon(new(big.Int)) && s.Random != nil {
		random := common.BigToHash(s.Random.Convert())
		return &random
	}
	return nil
}

func (s *stBlockEnvironment) GetFork() string {
	return s.fork
}

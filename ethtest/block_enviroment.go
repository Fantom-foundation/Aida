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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
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
	genesis       core.Genesis
	ctx           vm.BlockContext
}

func (s *stBlockEnvironment) GetCoinbase() common.Address {
	return s.ctx.Coinbase
}

func (s *stBlockEnvironment) GetBlobBaseFee() *big.Int {
	return s.ctx.BlobBaseFee
}

func (s *stBlockEnvironment) GetDifficulty() *big.Int {
	return s.ctx.Difficulty
}

func (s *stBlockEnvironment) GetGasLimit() uint64 {
	return s.ctx.GasLimit
}

func (s *stBlockEnvironment) GetNumber() uint64 {
	return s.ctx.BlockNumber.Uint64()
}

func (s *stBlockEnvironment) GetTimestamp() uint64 {
	return s.ctx.Time
}

func (s *stBlockEnvironment) GetBlockHash(blockNum uint64) (common.Hash, error) {
	return core.GetHashFn(s.genesis.ToBlock().Header(), nil)(blockNum), nil
}

func (s *stBlockEnvironment) GetBaseFee() *big.Int {
	return s.ctx.BaseFee
}

func (s *stBlockEnvironment) GetRandom() *common.Hash {
	return s.ctx.Random
}

func (s *stBlockEnvironment) GetChainConfig() *params.ChainConfig {
	return s.genesis.Config
}

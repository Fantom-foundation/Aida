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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
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
	genesis       core.Genesis
}

func (s *stBlockEnvironment) GetCoinbase() common.Address {
	return s.Coinbase
}

func (s *stBlockEnvironment) GetBlobBaseFee() *big.Int {
	return eip4844.CalcBlobFee(s.ExcessBlobGas.Uint64())
}

func (s *stBlockEnvironment) GetDifficulty() *big.Int {
	return s.Difficulty.Convert()
}

func (s *stBlockEnvironment) GetGasLimit() uint64 {
	return s.GasLimit.Uint64()
}

func (s *stBlockEnvironment) GetNumber() uint64 {
	return s.blockNumber
}

func (s *stBlockEnvironment) GetTimestamp() uint64 {
	return s.Timestamp.Uint64()
}

func (s *stBlockEnvironment) GetBlockHash(blockNumber uint64) (common.Hash, error) {
	return common.Hash{}, nil
}

func (s *stBlockEnvironment) GetBaseFee() *big.Int {
	return s.BaseFee.Convert()
}

// todo remove and redo into getters
func (s *stBlockEnvironment) GetBlockContext(*error) *vm.BlockContext {
	var baseFee *big.Int
	if s.genesis.Config.IsLondon(new(big.Int)) {
		baseFee = s.BaseFee.Convert()
		if baseFee == nil {
			// Retesteth uses `0x10` for genesis baseFee. Therefore, it defaults to
			// parent - 2 : 0xa as the basefee for 'this' context.
			baseFee = big.NewInt(0x0a)
		}
	}

	block := s.genesis.ToBlock()

	context := core.NewEVMBlockContext(block.Header(), nil, &s.Coinbase)
	context.GetHash = vmTestBlockHash
	context.BaseFee = baseFee
	context.Random = nil
	if s.Difficulty != nil {
		context.Difficulty = new(big.Int).Set(s.Difficulty.Convert())
	}
	if s.genesis.Config.IsLondon(new(big.Int)) && s.Random != nil {
		rnd := common.BigToHash(s.Random.Convert())
		context.Random = &rnd
		context.Difficulty = big.NewInt(0)
	}
	if s.genesis.Config.IsCancun(new(big.Int), block.Time()) && s.ExcessBlobGas != nil {
		context.BlobBaseFee = eip4844.CalcBlobFee(s.ExcessBlobGas.Uint64())
	}

	return &context
}

func (s *stBlockEnvironment) GetChainConfig() *params.ChainConfig {
	return s.genesis.Config
}

func vmTestBlockHash(n uint64) common.Hash {
	return common.BytesToHash(crypto.Keccak256([]byte(big.NewInt(int64(n)).String())))
}

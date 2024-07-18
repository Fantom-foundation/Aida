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

package txcontext

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// BlockEnvironment represents an interface for retrieving and modifying Ethereum-like blockchain environment information.
type BlockEnvironment interface {
	// GetCoinbase returns the coinbase address.
	GetCoinbase() common.Address

	// GetDifficulty returns the current difficulty level.
	GetDifficulty() *big.Int

	// GetGasLimit returns the maximum amount of gas that can be used in a block.
	GetGasLimit() uint64

	// GetNumber returns the current block number.
	GetNumber() uint64

	// GetTimestamp returns the timestamp of the current block.
	GetTimestamp() uint64

	// GetBlockHash returns the hash of the block with the given number.
	GetBlockHash(blockNumber uint64) (common.Hash, error)

	// GetBaseFee returns the base fee for transactions in the current block.
	GetBaseFee() *big.Int

	// GetBlockContext prepares a BlockContext. Any error produced by hashing should be passed to hashErr.
	GetBlockContext(hashErr *error) *vm.BlockContext

	GetChainConfig() *params.ChainConfig
}

// PrepareBlockCtx creates a block context for evm call from given BlockEnvironment.
// This func servers as a dummy as most of GetBlockContext() implementations
// share common block context preparation.
func PrepareBlockCtx(inputEnv BlockEnvironment, hashError *error) *vm.BlockContext {
	getHash := func(num uint64) common.Hash {
		var h common.Hash
		h, *hashError = inputEnv.GetBlockHash(num)
		return h
	}

	blockCtx := &vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    inputEnv.GetCoinbase(),
		BlockNumber: new(big.Int).SetUint64(inputEnv.GetNumber()),
		Time:        inputEnv.GetTimestamp(),
		Difficulty:  inputEnv.GetDifficulty(),
		GasLimit:    inputEnv.GetGasLimit(),
		GetHash:     getHash,
	}
	// If currentBaseFee is defined, add it to the vmContext.
	baseFee := inputEnv.GetBaseFee()
	if baseFee != nil {
		blockCtx.BaseFee = new(big.Int).Set(baseFee)
	}
	return blockCtx
}

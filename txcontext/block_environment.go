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
}

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

package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

func NewChainConduit(isEthereum bool, chainConfig *params.ChainConfig) *ChainConduit {
	return &ChainConduit{
		isEthereum:  isEthereum,
		chainConfig: chainConfig,
	}
}

// ChainConduit is used to determine special behaviour between Opera and Ethereum (and their hard forks different behaviours) (e.g. EndTransaction).
type ChainConduit struct {
	isEthereum  bool
	chainConfig *params.ChainConfig
}

func (c *ChainConduit) IsFinalise(block uint64) bool {
	b := big.NewInt(int64(block))
	if !c.isEthereum {
		return true
	} else {
		return c.chainConfig.IsByzantium(b)
	}
}

func (c *ChainConduit) DeleteEmptyObjects(block uint64) bool {
	if !c.isEthereum {
		return true
	} else {
		b := big.NewInt(int64(block))
		bz := c.chainConfig.IsByzantium(b)
		ei := c.chainConfig.IsEIP158(b)
		return bz || ei
	}
}

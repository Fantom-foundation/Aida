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

package substate

import (
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func NewBlockEnvironment(env *substate.Env, chainCfg *params.ChainConfig) txcontext.BlockEnvironment {
	return &blockEnvironment{env, chainCfg}
}

type blockEnvironment struct {
	*substate.Env
	chainCfg *params.ChainConfig
}

func (e *blockEnvironment) GetChainConfig() *params.ChainConfig {
	return e.chainCfg
}

func (e *blockEnvironment) GetBlockHash(block uint64) (common.Hash, error) {
	if e.BlockHashes == nil {
		return common.Hash{}, fmt.Errorf("getHash(%d) invoked, no blockhashes provided", block)
	}
	h, ok := e.BlockHashes[block]
	if !ok {
		return common.Hash(h), fmt.Errorf("getHash(%d) invoked, blockhash for that block not provided", block)
	}
	return common.Hash(h), nil
}

func (e *blockEnvironment) GetCoinbase() common.Address {
	return common.Address(e.Coinbase)
}

func (e *blockEnvironment) GetDifficulty() *big.Int {
	return e.Difficulty
}

func (e *blockEnvironment) GetGasLimit() uint64 {
	return e.GasLimit
}

func (e *blockEnvironment) GetNumber() uint64 {
	return e.Number
}

func (e *blockEnvironment) GetTimestamp() uint64 {
	return e.Timestamp
}

func (e *blockEnvironment) GetBaseFee() *big.Int {
	return e.BaseFee
}

func (e *blockEnvironment) GetBlockContext(hashErr *error) *vm.BlockContext {
	return txcontext.PrepareBlockCtx(e, hashErr)
}

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

package newsubstate

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

func NewTxContext(data *substate.Substate) txcontext.TxContext {
	return &substateData{data}
}

type substateData struct {
	*substate.Substate
}

func (t *substateData) GetStateHash() common.Hash {
	return common.Hash{}
}

func (t *substateData) GetInputState() txcontext.WorldState {
	return NewWorldState(t.InputAlloc)
}

func (t *substateData) GetOutputState() txcontext.WorldState {
	return NewWorldState(t.OutputAlloc)
}

func (t *substateData) GetBlockEnvironment() txcontext.BlockEnvironment {
	return NewBlockEnvironment(t.Env)
}

func (t *substateData) GetMessage() core.Message {
	var list types.AccessList
	for _, tuple := range t.Message.AccessList {
		var keys []common.Hash
		for _, key := range tuple.StorageKeys {
			keys = append(keys, common.Hash(key))
		}
		list = append(list, types.AccessTuple{Address: common.Address(tuple.Address), StorageKeys: keys})
	}
	return types.NewMessage(common.Address(t.Message.From), (*common.Address)(t.Message.To), t.Message.Nonce, t.Message.Value, t.Message.Gas, t.Message.GasPrice, t.Message.GasFeeCap, t.Message.GasTipCap, t.Message.Data, list, !t.Message.CheckNonce)
}

func (t *substateData) GetResult() txcontext.Result {
	return NewReceipt(t.Result)
}

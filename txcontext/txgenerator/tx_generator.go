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

package txgenerator

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

func NewTxContext(env txcontext.BlockEnvironment, msg core.Message) txcontext.TxContext {
	return &txData{Env: env, Message: msg}
}

type txData struct {
	txcontext.NilTxContext
	Env     txcontext.BlockEnvironment
	Message core.Message
}

func (t *txData) GetStateHash() common.Hash {
	// ignored
	return common.Hash{}
}

func (t *txData) GetBlockEnvironment() txcontext.BlockEnvironment {
	return t.Env
}

func (t *txData) GetMessage() core.Message {
	return t.Message
}

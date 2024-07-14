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
	"fmt"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

func newStateTestTxContest(stJson *stJSON, env *stBlockEnvironment, msg *core.Message, rootHash common.Hash, fork string, number int) txcontext.TxContext {
	return &stateTestContext{
		Fork:       fork,
		Label:      stJson.label,
		number:     number,
		env:        env,
		inputState: stJson.Pre,
		msg:        msg,
		rootHash:   rootHash,
	}
}

type stateTestContext struct {
	txcontext.NilTxContext
	Fork, Label string
	number      int
	env         *stBlockEnvironment
	inputState  types.GenesisAlloc
	msg         *core.Message
	rootHash    common.Hash
}

func (s *stateTestContext) GetStateHash() common.Hash {
	return s.rootHash
}

func (s *stateTestContext) GetOutputState() txcontext.WorldState {
	// we dont execute pseudo transactions here
	return nil
}

func (s *stateTestContext) GetInputState() txcontext.WorldState {
	return NewWorldState(s.inputState)
}

func (s *stateTestContext) GetBlockEnvironment() txcontext.BlockEnvironment {
	return s.env
}

func (s *stateTestContext) GetMessage() *core.Message {
	return s.msg
}

func (s *stateTestContext) String() string {
	return fmt.Sprintf("Test label: %v\nFork: %v\nTest Number: %v", s.Label, s.Fork, s.number)
}

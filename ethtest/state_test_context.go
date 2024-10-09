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
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func newStateTestTxContext(stJson *stJSON, msg *core.Message, post stPost, chainCfg *params.ChainConfig, fork string, postNumber int) txcontext.TxContext {
	return &StateTestContext{
		fork:          fork,
		path:          stJson.path,
		postNumber:    postNumber,
		description:   stJson.description,
		env:           stJson.CreateEnv(chainCfg, fork),
		inputState:    stJson.Pre,
		msg:           msg,
		rootHash:      post.RootHash,
		expectedError: post.ExpectException,
		txBytes:       post.TxBytes,
		logHash:       post.LogsHash,
	}
}

type StateTestContext struct {
	txcontext.NilTxContext
	fork          string // which fork is the test running
	path          string // path to file from which is the test
	description   string // description from JSON test file
	postNumber    int    // the post number within one 'fork' within one JSON file
	env           *stBlockEnvironment
	inputState    types.GenesisAlloc
	msg           *core.Message
	rootHash      common.Hash // expected root hash
	expectedError string      // expected error by processor
	logHash       common.Hash // expected rlp log hash
	txBytes       hexutil.Bytes
}

func (s *StateTestContext) GetTxBytes() hexutil.Bytes {
	return s.txBytes
}

func (s *stateTestContext) GetLogsHash() common.Hash {
	return s.logHash
}

func (s *StateTestContext) GetStateHash() common.Hash {
	return s.rootHash
}

func (s *StateTestContext) GetOutputState() txcontext.WorldState {
	// we dont execute pseudo transactions here
	return nil
}

func (s *StateTestContext) GetInputState() txcontext.WorldState {
	return NewWorldState(s.inputState)
}

func (s *StateTestContext) GetBlockEnvironment() txcontext.BlockEnvironment {
	return s.env
}

func (s *StateTestContext) GetMessage() *core.Message {
	return s.msg
}

func (s *StateTestContext) GetResult() txcontext.Result {
	return stateTestResult{s.expectedError}
}

func (s *StateTestContext) String() string {
	return fmt.Sprintf("Test path: %v\nDescription: %v\nFork: %v\nPost number: %v", s.path, s.description, s.fork, s.postNumber)
}

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
	"github.com/ethereum/go-ethereum/params"
)

func newStateTestTxContext(
	stJson *stJSON,
	msg *core.Message,
	post stPost,
	chainCfg *params.ChainConfig,
	testLabel string,
	fork string,
	postNumber int,
) txcontext.TxContext {
	return &stateTestContext{
		path:          stJson.path,
		testLabel:     testLabel,
		fork:          fork,
		postNumber:    postNumber,
		env:           stJson.CreateEnv(chainCfg, fork),
		inputState:    stJson.Pre,
		msg:           msg,
		rootHash:      post.RootHash,
		expectedError: post.ExpectException,
	}
}

type stateTestContext struct {
	txcontext.NilTxContext
	path          string // path to file from which is the test
	testLabel     string // the test label within one JSON file (key to the JSON)
	fork          string // which fork is the test running
	postNumber    int    // the post number within one 'fork' within one JSON file
	env           *stBlockEnvironment
	inputState    types.GenesisAlloc
	msg           *core.Message
	rootHash      common.Hash
	expectedError string
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

func (s *stateTestContext) GetResult() txcontext.Result {
	return stateTestResult{s.expectedError}
}

func (s *stateTestContext) String() string {
	return fmt.Sprintf(
		"Test path: %v\n"+
			"Test label: %v\n"+
			"Fork: %v\n"+
			"PostNumber: %v\n", s.path, s.testLabel, s.fork, s.postNumber)
}

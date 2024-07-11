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
	"strings"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

// TODO: This should only work as a unmarshaller and should not be used as transaction, new struct should be created for this regard
type StJSON struct {
	txcontext.NilTxContext
	FilePath     string
	TestLabel    string
	UsedNetwork  string
	Env          stEnv               `json:"env"`
	Pre          types.GenesisAlloc  `json:"pre"`
	Tx           stTransaction       `json:"transaction"`
	Out          hexutil.Bytes       `json:"out"`
	Post         map[string][]stPost `json:"post"`
	currentIndex int
}

func (s *StJSON) GetStateHash() common.Hash {
	for _, n := range usableForks {
		if p, ok := s.Post[n]; ok {
			return p[0].RootHash
		}
	}

	// this cannot happen because we only allow usable tests
	return common.Hash{}
}

func (s *StJSON) GetOutputState() txcontext.WorldState {
	// we dont execute pseudo transactions here
	return nil
}

func (s *StJSON) GetInputState() txcontext.WorldState {
	return NewWorldState(s.Pre)
}

func (s *StJSON) GetBlockEnvironment() txcontext.BlockEnvironment {
	return &s.Env
}

func (s *StJSON) GetMessage() *core.Message {
	baseFee := s.Env.BaseFee
	if baseFee == nil {
		// ethereum uses `0x10` for genesis baseFee. Therefore, it defaults to
		// parent - 2 : 0xa as the basefee for 'this' context.
		baseFee = &BigInt{*big.NewInt(0x0a)}
	}

	msg, err := s.Tx.toMessage(s.getTransactionData(), baseFee)

	if err != nil {
		panic(err)
	}

	return msg
}

func (s *StJSON) getTransactionData() stPost {
	return s.Post[s.UsedNetwork][0]
}

// stPost indicates data for each transaction.
type stPost struct {
	// RootHash holds expected state hash after a transaction is executed.
	RootHash common.Hash `json:"hash"`
	// LogsHash holds expected logs hash (Bloom) after a transaction is executed.
	LogsHash        common.Hash   `json:"logs"`
	TxBytes         hexutil.Bytes `json:"txbytes"`
	ExpectException string        `json:"expectException"`
	indexes         Index
}

// Index indicates position of data, gas, value for executed transaction.
type Index struct {
	Data  int `json:"data"`
	Gas   int `json:"gas"`
	Value int `json:"value"`
}

// Divide iterates usableForks and validation data in ETH JSON State tests and creates test for each fork
func (s *StJSON) Divide(chainId utils.ChainID) (dividedTests []*StJSON) {
	// each test contains multiple validation data for different forks.
	// we create a test for each usable fork
	for _, fork := range usableForks {
		var test StJSON
		if posts, ok := s.Post[fork]; ok {
			test = *s               // copy all the test data
			test.UsedNetwork = fork // add correct fork name
			// add block number to env (+1 just to make sure we are within wanted fork)
			test.Env.blockNumber = utils.KeywordBlocks[chainId][strings.ToLower(fork)] + 1

			// iterate over all posts (each transaction data) and execute tx for each one
			for i, _ := range posts {
				test.currentIndex = i
				dividedTests = append(dividedTests, &test)
			}

		}
	}

	return dividedTests
}

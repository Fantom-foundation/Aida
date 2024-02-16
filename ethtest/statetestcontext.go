package ethtest

import (
	"math/big"
	"strings"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
)

type StJSON struct {
	txcontext.NilTxContext
	TestLabel   string
	UsedNetwork string
	Env         stEnv                    `json:"env"`
	Pre         core.GenesisAlloc        `json:"pre"`
	Tx          stTransaction            `json:"transaction"`
	Out         hexutil.Bytes            `json:"out"`
	Post        map[string][]stPostState `json:"post"`
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

func (s *StJSON) GetMessage() core.Message {
	baseFee := s.Env.BaseFee
	if baseFee == nil {
		// ethereum uses `0x10` for genesis baseFee. Therefore, it defaults to
		// parent - 2 : 0xa as the basefee for 'this' context.
		baseFee = &BigInt{*big.NewInt(0x0a)}
	}

	msg, err := s.Tx.toMessage(s.getPostState(), baseFee)

	if err != nil {
		panic(err)
	}

	return msg
}

func (s *StJSON) getPostState() stPostState {
	return s.Post[s.UsedNetwork][0]
}

type stPostState struct {
	RootHash        common.Hash   `json:"hash"`
	LogsHash        common.Hash   `json:"logs"`
	TxBytes         hexutil.Bytes `json:"txbytes"`
	ExpectException string        `json:"expectException"`
	indexes         Index
}

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
		if _, ok := s.Post[fork]; ok {
			test = *s               // copy all the test data
			test.UsedNetwork = fork // add correct fork name

			// add block number to env (+1 just to make sure we are within wanted fork)
			test.Env.blockNumber = utils.KeywordBlocks[chainId][strings.ToLower(fork)] + 1
			dividedTests = append(dividedTests, &test)
		}
	}

	return dividedTests
}

package ethtest

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/tests"
)

// stJSON serves as a 'middleman' into which are data unmarshalled from geth test files.
type stJSON struct {
	path        string
	description string
	Env         stBlockEnvironment  `json:"env"`
	Pre         types.GenesisAlloc  `json:"pre"`
	Tx          stTransaction       `json:"transaction"`
	Out         hexutil.Bytes       `json:"out"`
	Post        map[string][]stPost `json:"post"`
}

func (s *stJSON) setPath(path string) {
	s.path = path
}

func (s *stJSON) setDescription(desc string) {
	s.description = desc
}

func (s *stJSON) CreateEnv(fork string) (*stBlockEnvironment, error) {
	config, _, err := tests.GetChainConfig(fork)
	if err != nil {
		return nil, fmt.Errorf("cannot get chain config: %w", err)
	}

	// Create copy as each tx needs its own env
	env := s.Env
	// todo remove genesis, check:

	env.genesis = core.Genesis{
		Config:     config,
		Coinbase:   env.Coinbase,
		Difficulty: env.Difficulty.Convert(),
		GasLimit:   env.GasLimit.Uint64(),
		Number:     env.Number.Uint64(),
		Timestamp:  env.Timestamp.Uint64(),
		Alloc:      s.Pre,
	}

	if env.Random != nil {
		env.genesis.Mixhash = common.BigToHash(env.Random.Convert())
		env.genesis.Difficulty = big.NewInt(0)
	}

	env.ctx = core.NewEVMBlockContext(env.genesis.ToBlock().Header(), nil, &s.Env.Coinbase)

	return &env, nil
}

// stPost indicates data for each transaction.
type stPost struct {
	// RootHash holds expected state hash after a transaction is executed.
	RootHash common.Hash `json:"hash"`
	// LogsHash holds expected logs hash (Bloom) after a transaction is executed.
	LogsHash        common.Hash   `json:"logs"`
	TxBytes         hexutil.Bytes `json:"txbytes"`
	ExpectException string        `json:"expectException"`
	Indexes         Index         `json:"indexes"`
}

// Index indicates position of data, gas, value for executed transaction.
type Index struct {
	Data  int `json:"data"`
	Gas   int `json:"gas"`
	Value int `json:"value"`
}

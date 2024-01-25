package ethtest

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// OpenStateTests opens
func OpenStateTests(path string) ([]*stJSON, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var tests []*stJSON

	if info.IsDir() {
		// todo iterating all files always fail validation
		tests, err = GetTestsWithinPath[*stJSON](path, StateTests)
		if err != nil {
			return nil, err
		}

		// todo only one test was found to work located in GeneralStateTests/stBugs/evmBytecode.json
		// although this test does not work when iterating all tests

	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		byteJSON, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}

		var b map[string]*stJSON
		err = json.Unmarshal(byteJSON, &b)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal file %v", path)
		}

		for _, t := range b {
			tests = append(tests, t)
		}
	}

	return tests, nil
}

type stJSON struct {
	txcontext.NilTxContext
	usedNetwork string
	Env         stEnv                    `json:"env"`
	Pre         core.GenesisAlloc        `json:"pre"`
	Tx          stTransaction            `json:"transaction"`
	Out         hexutil.Bytes            `json:"out"`
	Post        map[string][]stPostState `json:"post"`
}

func (s *stJSON) GetStateHash() common.Hash {
	for _, n := range usableForks {
		if p, ok := s.Post[n]; ok {
			return p[0].RootHash
		}
	}

	// this cannot happen because we only allow usable tests
	return common.Hash{}
}

func (s *stJSON) GetOutputState() txcontext.WorldState {
	// we dont execute pseudo transactions here
	return nil
}

func (s *stJSON) GetInputState() txcontext.WorldState {
	return NewGethWorldState(s.Pre)
}

func (s *stJSON) GetBlockEnvironment() txcontext.BlockEnvironment {
	return &s.Env
}

func (s *stJSON) GetMessage() core.Message {
	baseFee := s.Env.BaseFee
	if baseFee == nil {
		// Retesteth uses `0x10` for genesis baseFee. Therefore, it defaults to
		// parent - 2 : 0xa as the basefee for 'this' context.
		baseFee = &BigInt{*big.NewInt(0x0a)}
	}

	msg, err := s.Tx.toMessage(s.getPostState(), baseFee)

	if err != nil {
		panic(err)
	}

	return msg
}

func (s *stJSON) getPostState() stPostState {
	return s.Post[s.usedNetwork][0]
}

// Divide iterates usableForks and validation data in ETH JSON State tests and creates test for each fork
func (s *stJSON) Divide(chainId utils.ChainID) (dividedTests []*stJSON) {
	// each test contains multiple validation data for different forks.
	// we create a test for each usable fork

	for _, fork := range usableForks {
		var test stJSON
		if _, ok := s.Post[fork]; ok {
			test = *s               // copy all the test data
			test.usedNetwork = fork // add correct fork name

			// add block number to env (+1 just to make sure we are within wanted fork)
			test.Env.blockNumber = utils.KeywordBlocks[chainId][fork] + 1
			dividedTests = append(dividedTests, &test)
		}
	}

	return dividedTests
}

type stTransaction struct {
	GasPrice             *BigInt             `json:"gasPrice"`
	MaxFeePerGas         *BigInt             `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *BigInt             `json:"maxPriorityFeePerGas"`
	Nonce                *BigInt             `json:"nonce"`
	To                   string              `json:"to"`
	Data                 []string            `json:"data"`
	AccessLists          []*types.AccessList `json:"accessLists,omitempty"`
	GasLimit             []*BigInt           `json:"gasLimit"`
	Value                []string            `json:"value"`
	PrivateKey           hexutil.Bytes       `json:"secretKey"`
}

func (tx *stTransaction) toMessage(ps stPostState, baseFee *BigInt) (*types.Message, error) {
	// Derive sender from private key if present.
	var from common.Address
	if len(tx.PrivateKey) > 0 {
		key, err := crypto.ToECDSA(tx.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid private key: %v", err)
		}
		from = crypto.PubkeyToAddress(key.PublicKey)
	}
	// Parse recipient if present.
	var to *common.Address
	if tx.To != "" {
		to = new(common.Address)
		if err := to.UnmarshalText([]byte(tx.To)); err != nil {
			return nil, fmt.Errorf("invalid to address: %v", err)
		}
	}

	// Get values specific to this post state.
	if ps.indexes.Data > len(tx.Data) {
		return nil, fmt.Errorf("tx data index %d out of bounds", ps.indexes.Data)
	}
	if ps.indexes.Value > len(tx.Value) {
		return nil, fmt.Errorf("tx value index %d out of bounds", ps.indexes.Value)
	}
	if ps.indexes.Gas > len(tx.GasLimit) {
		return nil, fmt.Errorf("tx gas limit index %d out of bounds", ps.indexes.Gas)
	}
	dataHex := tx.Data[ps.indexes.Data]
	valueHex := tx.Value[ps.indexes.Value]
	gasLimit := tx.GasLimit[ps.indexes.Gas]
	// Value, Data hex encoding is messy: https://github.com/ethereum/tests/issues/203
	value := new(big.Int)
	if valueHex != "0x" {
		v, ok := math.ParseBig256(valueHex)
		if !ok {
			return nil, fmt.Errorf("invalid tx value %q", valueHex)
		}
		value = v
	}
	data, err := hex.DecodeString(strings.TrimPrefix(dataHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid tx data %q", dataHex)
	}
	var accessList types.AccessList
	if tx.AccessLists != nil && tx.AccessLists[ps.indexes.Data] != nil {
		accessList = *tx.AccessLists[ps.indexes.Data]
	}
	// If baseFee provided, set gasPrice to effectiveGasPrice.
	gasPrice := tx.GasPrice
	if baseFee != nil {
		if tx.MaxFeePerGas == nil {
			tx.MaxFeePerGas = gasPrice
		}
		if tx.MaxFeePerGas == nil {
			tx.MaxFeePerGas = new(BigInt)
		}
		if tx.MaxPriorityFeePerGas == nil {
			tx.MaxPriorityFeePerGas = tx.MaxFeePerGas
		}
		gasPrice = &BigInt{*math.BigMin(new(big.Int).Add(tx.MaxPriorityFeePerGas.Convert(), baseFee.Convert()),
			tx.MaxFeePerGas.Convert())}
	}
	if gasPrice == nil {
		return nil, fmt.Errorf("no gas price provided")
	}

	msg := types.NewMessage(from, to, tx.Nonce.Uint64(), value, gasLimit.Uint64(), gasPrice.Convert(),
		tx.MaxFeePerGas.Convert(), tx.MaxPriorityFeePerGas.Convert(), data, accessList, false)
	return &msg, nil
}

type stEnv struct {
	blockNumber uint64
	Coinbase    common.Address `json:"currentCoinbase"   gencodec:"required"`
	Difficulty  *BigInt        `json:"currentDifficulty" gencodec:"required"`
	GasLimit    *BigInt        `json:"currentGasLimit"   gencodec:"required"`
	Number      *BigInt        `json:"currentNumber"     gencodec:"required"`
	Timestamp   *BigInt        `json:"currentTimestamp"  gencodec:"required"`
	BaseFee     *BigInt        `json:"currentBaseFee"  gencodec:"optional"`
}

func (s *stEnv) GetCoinbase() common.Address {
	return s.Coinbase
}

func (s *stEnv) GetDifficulty() *big.Int {
	return s.Difficulty.Convert()
}

func (s *stEnv) GetGasLimit() uint64 {
	return s.GasLimit.Uint64()
}

func (s *stEnv) GetNumber() uint64 {
	//return s.Number.Uint64() todo i think this is not correct, json has always number 1
	return s.blockNumber
}

func (s *stEnv) GetTimestamp() uint64 {
	return s.Timestamp.Uint64()
}

func (s *stEnv) GetBlockHash(blockNumber uint64) common.Hash {
	return common.Hash{}
}

func (s *stEnv) GetBaseFee() *big.Int {
	return s.BaseFee.Convert()
}

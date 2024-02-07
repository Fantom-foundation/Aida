package ethtest

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"testing"

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
func OpenStateTests(path string) ([]*StJSON, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var tests []*StJSON

	if info.IsDir() {
		tests, err = GetTestsWithinPath[*StJSON](path, StateTests)
		if err != nil {
			return nil, err
		}

	} else {
		tests, err = readTestsFromFile(path)
		if err != nil {
			return nil, err
		}
	}

	return tests, nil
}

func readTestsFromFile(path string) ([]*StJSON, error) {
	var tests []*StJSON
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	byteJSON, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var b map[string]*StJSON
	err = json.Unmarshal(byteJSON, &b)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal file %v", path)
	}

	testLabel := getTestLabel(path)

	for _, t := range b {
		t.TestLabel = testLabel
		tests = append(tests, t)
	}
	return tests, nil
}

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

func CreateTestDataFile(t *testing.T) string {
	path := t.TempDir()
	pathFile := path + "/test.json"
	stData := CreateTestData(t)

	jsonData, err := json.Marshal(stData)
	if err != nil {
		t.Errorf("Marshal() error = %v, wantErr %v", err, nil)
	}

	jsonStr := "{ \"test\" : " + string(jsonData) + "}"

	jsonData = []byte(jsonStr)
	// Initialize pathFile
	err = os.WriteFile(pathFile, jsonData, 0644)
	if err != nil {
		t.Errorf("WriteFile() error = %v, wantErr %v", err, nil)
	}
	return pathFile
}

func CreateTestData(t *testing.T) *StJSON {
	bInt := new(big.Int).SetUint64(1)
	return &StJSON{
		TestLabel:   "TestLabel",
		UsedNetwork: "Istanbul",
		Env: stEnv{
			blockNumber: 1,
			Coinbase:    common.Address{},
			Difficulty:  &BigInt{*bInt},
			GasLimit:    &BigInt{*bInt},
			Number:      &BigInt{*bInt},
			Timestamp:   &BigInt{*bInt},
			BaseFee:     &BigInt{*bInt},
		},
		Pre: core.GenesisAlloc{
			common.HexToAddress("0x1"): core.GenesisAccount{
				Balance: big.NewInt(1000),
				Nonce:   1,
			},
			common.HexToAddress("0x2"): core.GenesisAccount{
				Balance: big.NewInt(2000),
				Nonce:   2,
			},
		},
		Tx: stTransaction{
			GasPrice:             &BigInt{*bInt},
			MaxFeePerGas:         &BigInt{*bInt},
			MaxPriorityFeePerGas: &BigInt{*bInt},
			Nonce:                &BigInt{*bInt},
			To:                   common.HexToAddress("0x10").Hex(),
			Data:                 []string{"0x"},
			GasLimit:             []*BigInt{{*bInt}},
			Value:                []string{"0x00"},
			PrivateKey:           hexutil.MustDecode("0x45a915e4d060149eb4365960e6a7a45f334393093061116b197e3240065ff2d8"),
		},
		Post: map[string][]stPostState{
			"Istanbul": {
				{
					RootHash: common.HexToHash("0x20"),
					LogsHash: common.HexToHash("0x30"),
					indexes:  Index{},
				},
			},
		},
		//Out: hexutil.Bytes("0x0"),
	}
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

func (tx *stTransaction) MarshalJSON() ([]byte, error) {
	gasLimitArr := make([]string, len(tx.GasLimit))
	for i, v := range tx.GasLimit {
		gasLimitArr[i] = "0x" + v.Text(16)
	}

	// Create a new data structure that has the same fields as stTransaction but is a valid format for json.Marshal
	type Alias stTransaction
	return json.Marshal(&struct {
		GasPrice             string   `json:"gasPrice"`
		MaxFeePerGas         string   `json:"maxFeePerGas"`
		MaxPriorityFeePerGas string   `json:"maxPriorityFeePerGas"`
		Nonce                string   `json:"nonce"`
		GasLimit             []string `json:"gasLimit"`
		*Alias
	}{
		GasPrice:             "0x" + tx.GasPrice.Text(16),
		MaxFeePerGas:         "0x" + tx.MaxFeePerGas.Text(16),
		MaxPriorityFeePerGas: "0x" + tx.MaxPriorityFeePerGas.Text(16),
		Nonce:                "0x" + tx.Nonce.Text(16),
		GasLimit:             gasLimitArr,
		Alias:                (*Alias)(tx),
	})
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

func (env *stEnv) MarshalJSON() ([]byte, error) {
	// Create a new data structure that has the same fields as stEnv but is a valid format for json.Marshal
	type Alias stEnv
	return json.Marshal(&struct {
		Difficulty string `json:"currentDifficulty"`
		GasLimit   string `json:"currentGasLimit"`
		Number     string `json:"currentNumber"`
		Timestamp  string `json:"currentTimestamp"`
		BaseFee    string `json:"currentBaseFee"`
		*Alias
	}{
		Difficulty: "0x" + env.Difficulty.Text(16),
		GasLimit:   "0x" + env.GasLimit.Text(16),
		Number:     "0x" + env.Number.Text(16),
		Timestamp:  "0x" + env.Timestamp.Text(16),
		BaseFee:    "0x" + env.BaseFee.Text(16),
		Alias:      (*Alias)(env),
	})
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
	return s.blockNumber
}

func (s *stEnv) GetTimestamp() uint64 {
	return s.Timestamp.Uint64()
}

func (s *stEnv) GetBlockHash(blockNumber uint64) (common.Hash, error) {
	return common.Hash{}, nil
}

func (s *stEnv) GetBaseFee() *big.Int {
	return s.BaseFee.Convert()
}

// getTestLabel returns the last folder name and the filename of the given path
func getTestLabel(path string) string {
	// Split the path into components
	pathComponents := strings.Split(path, "/")

	var lastFolderName = ""
	var filename = ""

	if len(pathComponents) > 1 {
		// Extract the last folder name
		lastFolderName = pathComponents[len(pathComponents)-2]
	}

	if len(pathComponents) > 0 {
		// Extract the filename
		filename = pathComponents[len(pathComponents)-1]
	}
	return lastFolderName + "/" + filename
}

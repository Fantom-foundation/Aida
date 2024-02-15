package blockchaintest

import (
	"encoding/json"
	"io"
	"os"

	"github.com/Fantom-foundation/Aida/ethtest/util"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func Open(path string) ([]*BtJSON, error) {
	fpaths, err := utils.GetDirectoryFiles(path)
	if err != nil {
		return nil, err
	}

	var (
		tests                                    []*BtJSON
		unmarshalled, nilBlocks, unusableNetwork uint64
	)

	for _, p := range fpaths {
		file, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		byteJSON, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}

		var b map[string]*BtJSON
		err = json.Unmarshal(byteJSON, &b)
		if err != nil {
			// todo try the other unmarshaling
			unmarshalled++
			// skip any unreadable tests
			continue
		}

		for _, t := range b {
			if t.Blocks == nil {
				nilBlocks++
				// skip any non block tests
				continue
			}

			if !t.isWithinUsableNetworks() {
				unusableNetwork++
				continue
			}
			tests = append(tests, t)
		}
	}

	return tests, nil
}

// todo this might not be correct
var usableNetworks = []string{"Istanbul, MuirGlacier", "Berlin", "London"}

type BtJSON struct {
	TestLabel  string
	Blocks     []BtBlock         `json:"blocks"`
	Genesis    BlockEnvironment  `json:"genesisBlockHeader"`
	Pre        core.GenesisAlloc `json:"pre"`
	Post       core.GenesisAlloc `json:"postState"`
	Network    string            `json:"network"`
	SealEngine string            `json:"sealEngine"`
}

func (t *BtJSON) SetLabel(label string) {
	t.TestLabel = label
}
func (t *BtJSON) isWithinUsableNetworks() bool {
	for _, network := range usableNetworks {
		if network == t.Network {
			return true
		}
	}

	return false
}

type BtBlock struct {
	TestLabel       string
	UsedNetwork     string
	BlockHeader     *BlockEnvironment `json:"blockHeader"`
	ExpectException string
	Rlp             string              `json:"rlp"`
	UncleHeaders    []*BlockEnvironment `json:"uncleHeaders"`
	Transactions    []*Transaction
}

func (bb *BtBlock) Decode() (*types.Block, error) {
	data, err := hexutil.Decode(bb.Rlp)
	if err != nil {
		return nil, err
	}
	var b types.Block
	err = rlp.DecodeBytes(data, &b)
	return &b, err
}

type Transaction struct {
	To         *common.Address
	From       common.Address `json:"sender"`
	Nonce      *util.BigInt
	GasLimit   *util.BigInt
	GasPrice   *util.BigInt
	GasFeeCap  *util.BigInt
	GasTipCap  *util.BigInt
	Data       string
	AccessList types.AccessList
	Value      *util.BigInt
	Amount     *util.BigInt
	IsFake     bool
}

func (t Transaction) ToMessage() types.Message {
	return types.NewMessage(t.From, t.To, t.Nonce.Uint64(), t.Amount.Convert(), t.GasLimit.Uint64(), t.GasPrice.Convert(), t.GasFeeCap.Convert(), t.GasTipCap.Convert(), hexutil.MustDecode(t.Data), t.AccessList, t.IsFake)
}

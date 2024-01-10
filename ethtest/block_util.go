package ethtest

import (
	"encoding/json"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

type Data struct {
	Env     *Env
	Msg     types.Message
	Genesis *core.Genesis
	Post    core.GenesisAlloc
	Cfg     *Config
}

func NewData(block BtBlock, tx *Transaction, bt *BtJSON, chainID utils.ChainID) *Data {
	return &Data{
		Env:     block.BlockHeader,
		Msg:     tx.ToMessage(),
		Genesis: bt.CreateGenesis(utils.GetChainConfig(chainID)),
		Post:    bt.Post,
		Cfg: &Config{
			Network:    bt.Network,
			SealEngine: bt.SealEngine,
		},
	}
}

type Config struct {
	Network    string
	SealEngine string
}

func Open(path string) ([]*BtJSON, error) {
	file, err := os.Open(path)
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
		return nil, err
	}

	var tests []*BtJSON

	for _, t := range b {
		tests = append(tests, t)
	}

	return tests, nil
}

type EthTestStateAlloc struct {
	Nonce   uint64
	Balance *big.Int
	Storage map[common.Hash]common.Hash
	Code    []byte
}

type BtJSON struct {
	Blocks     []BtBlock         `json:"blocks"`
	Genesis    Env               `json:"genesisBlockHeader"`
	Pre        core.GenesisAlloc `json:"pre"`
	Post       core.GenesisAlloc `json:"postState"`
	Network    string            `json:"network"`
	SealEngine string            `json:"sealEngine"`
}

func (t *BtJSON) CreateGenesis(cfg *params.ChainConfig) *core.Genesis {
	return &core.Genesis{
		Config:     cfg,
		Nonce:      t.Genesis.Nonce.Uint64(),
		Timestamp:  t.Genesis.Timestamp.Uint64(),
		ParentHash: t.Genesis.ParentHash,
		ExtraData:  t.Genesis.ExtraData,
		GasLimit:   t.Genesis.GasLimit.Uint64(),
		GasUsed:    t.Genesis.GasUsed.Uint64(),
		Difficulty: t.Genesis.Difficulty.Convert(),
		Mixhash:    t.Genesis.MixHash,
		Coinbase:   t.Genesis.Coinbase,
		Alloc:      t.Pre,
		BaseFee:    t.Genesis.BaseFeePerGas.Convert(),
	}

}

type BtBlock struct {
	BlockHeader     *Env `json:"blockHeader"`
	ExpectException string
	Rlp             string `json:"rlp"`
	UncleHeaders    []*Env `json:"uncleHeaders"`
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
	Nonce      *BigInt
	GasLimit   *BigInt
	GasPrice   *BigInt
	GasFeeCap  *BigInt
	GasTipCap  *BigInt
	Data       string
	AccessList types.AccessList
	Value      *BigInt
	Amount     *BigInt
	IsFake     bool
}

func (t Transaction) ToMessage() types.Message {
	return types.NewMessage(t.From, t.To, t.Nonce.Uint64(), t.Amount.Convert(), t.GasLimit.Uint64(), t.GasPrice.Convert(), t.GasFeeCap.Convert(), t.GasTipCap.Convert(), hexutil.MustDecode(t.Data), t.AccessList, t.IsFake)
}

type Env struct {
	BaseFee          *BigInt `json:"baseFeePerGas"`
	Bloom            types.Bloom
	Coinbase         common.Address
	MixHash          common.Hash
	Nonce            types.BlockNonce
	Number           *BigInt
	Hash             common.Hash
	ParentHash       common.Hash
	ReceiptTrie      common.Hash
	StateRoot        common.Hash
	TransactionsTrie common.Hash
	UncleHash        common.Hash
	ExtraData        []byte
	Difficulty       *BigInt
	GasLimit         *BigInt
	GasUsed          *BigInt
	Timestamp        *BigInt
	BaseFeePerGas    *BigInt
}

type BigInt struct {
	big.Int
}

func (i *BigInt) Convert() *big.Int {
	if i == nil {
		return new(big.Int)
	}
	return &i.Int
}

func (i *BigInt) UnmarshalJSON(b []byte) error {
	var val string
	err := json.Unmarshal(b, &val)
	if err != nil {
		return err
	}

	i.SetString(strings.TrimPrefix(val, "0x"), 16)

	return nil
}

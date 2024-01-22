package blocktest

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type blockEnvironment struct {
	baseFee          *BigInt `json:"currentBaseFee"`
	bloom            types.Bloom
	coinbase         common.Address `json:"currentCoinbase"`
	mixHash          common.Hash
	nonce            types.BlockNonce
	number           *BigInt `json:"currentNumber"`
	hash             common.Hash
	parentHash       common.Hash
	receiptTrie      common.Hash
	stateRoot        common.Hash
	transactionsTrie common.Hash
	uncleHash        common.Hash
	extraData        []byte
	difficulty       *BigInt `json:"currentDifficulty"`
	gasLimit         *BigInt `json:"currentGasLimit"`
	gasUsed          *BigInt
	timestamp        *BigInt `json:"currentTimestamp"`
	baseFeePerGas    *BigInt
}

func (b *blockEnvironment) GetCoinbase() common.Address {
	return b.coinbase
}

func (b *blockEnvironment) GetDifficulty() *big.Int {
	return b.difficulty.Convert()
}

func (b *blockEnvironment) GetGasLimit() uint64 {
	return b.gasLimit.Uint64()
}

func (b *blockEnvironment) GetNumber() uint64 {
	return b.number.Uint64()
}

func (b *blockEnvironment) GetTimestamp() uint64 {
	return b.timestamp.Uint64()
}

func (b *blockEnvironment) GetBlockHash(uint64) common.Hash {
	return b.hash // todo is this correct?
}

func (b *blockEnvironment) GetBaseFee() *big.Int {
	return b.baseFee.Convert()
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

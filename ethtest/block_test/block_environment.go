package blocktest

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/ethtest/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type blockEnvironment struct {
	baseFee          *util.BigInt `json:"currentBaseFee"`
	bloom            types.Bloom
	coinbase         common.Address `json:"currentCoinbase"`
	mixHash          common.Hash
	nonce            types.BlockNonce
	number           *util.BigInt `json:"currentNumber"`
	hash             common.Hash
	parentHash       common.Hash
	receiptTrie      common.Hash
	stateRoot        common.Hash
	transactionsTrie common.Hash
	uncleHash        common.Hash
	extraData        []byte
	difficulty       *util.BigInt `json:"currentDifficulty"`
	gasLimit         *util.BigInt `json:"currentGasLimit"`
	gasUsed          *util.BigInt
	timestamp        *util.BigInt `json:"currentTimestamp"`
	baseFeePerGas    *util.BigInt
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

func (b *blockEnvironment) GetBlockHash(uint64) (common.Hash, error) {
	return b.hash, nil // todo is this correct?
}

func (b *blockEnvironment) GetBaseFee() *big.Int {
	return b.baseFee.Convert()
}

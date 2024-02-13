package blocktest

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/ethtest/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type BlockEnvironment struct {
	BaseFee          *util.BigInt //`json:"currentBaseFee"`
	Bloom            types.Bloom
	Coinbase         common.Address //`json:"currentCoinbase"`
	MixHash          common.Hash
	Nonce            types.BlockNonce
	Number           *util.BigInt //`json:"currentNumber"`
	Hash             common.Hash
	ParentHash       common.Hash
	ReceiptTrie      common.Hash
	StateRoot        common.Hash
	TransactionsTrie common.Hash
	UncleHash        common.Hash
	ExtraData        []byte
	Difficulty       *util.BigInt //`json:"currentDifficulty"`
	GasLimit         *util.BigInt //`json:"GasLimit"`
	GasUsed          *util.BigInt
	Timestamp        *util.BigInt //`json:"currentTimestamp"`
	BaseFeePerGas    *util.BigInt
}

func (b *BlockEnvironment) GetCoinbase() common.Address {
	return b.Coinbase
}

func (b *BlockEnvironment) GetDifficulty() *big.Int {
	return b.Difficulty.Convert()
}

func (b *BlockEnvironment) GetGasLimit() uint64 {
	return b.GasLimit.Uint64()
}

func (b *BlockEnvironment) GetNumber() uint64 {
	return b.Number.Uint64()
}

func (b *BlockEnvironment) GetTimestamp() uint64 {
	return b.Timestamp.Uint64()
}

func (b *BlockEnvironment) GetBlockHash(uint64) (common.Hash, error) {
	return b.Hash, nil // todo maybe use this instead of calculating hash in transaction_processor
}

func (b *BlockEnvironment) GetBaseFee() *big.Int {
	return b.BaseFee.Convert()
}

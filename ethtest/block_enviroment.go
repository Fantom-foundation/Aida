package ethtest

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

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

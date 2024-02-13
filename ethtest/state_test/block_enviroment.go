package state_test

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/ethtest/util"
	"github.com/ethereum/go-ethereum/common"
)

type stEnv struct {
	blockNumber uint64
	Coinbase    common.Address `json:"currentCoinbase"   gencodec:"required"`
	Difficulty  *util.BigInt   `json:"currentDifficulty" gencodec:"required"`
	GasLimit    *util.BigInt   `json:"currentGasLimit"   gencodec:"required"`
	Number      *util.BigInt   `json:"currentNumber"     gencodec:"required"`
	Timestamp   *util.BigInt   `json:"currentTimestamp"  gencodec:"required"`
	BaseFee     *util.BigInt   `json:"currentBaseFee"  gencodec:"optional"`
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

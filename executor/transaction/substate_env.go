package transaction

import (
	"math/big"

	substateCommon "github.com/Fantom-foundation/Substate/geth/common"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
)

func NewSubstateEnv(env *substate.Env) BlockEnvironment {
	return &substateEnv{env}
}

type substateEnv struct {
	*substate.Env
}

func (e *substateEnv) GetBlockHash(block uint64) common.Hash {
	return common.Hash(e.BlockHashes[block])
}

func (e *substateEnv) SetBlockHash(block uint64, hash common.Hash) {
	e.BlockHashes[block] = substateCommon.Hash(hash)
}

func (e *substateEnv) GetCoinbase() common.Address {
	return common.Address(e.Coinbase)
}

func (e *substateEnv) SetCoinbase(coinbase common.Address) {
	e.Coinbase = substateCommon.Address(coinbase)
}

func (e *substateEnv) GetDifficulty() *big.Int {
	return e.Difficulty
}

func (e *substateEnv) SetDifficulty(difficulty *big.Int) {
	e.Difficulty = difficulty
}

func (e *substateEnv) GetGasLimit() uint64 {
	return e.GasLimit
}

func (e *substateEnv) SetGasLimit(gasLimit uint64) {
	e.GasLimit = gasLimit
}

func (e *substateEnv) GetNumber() uint64 {
	return e.Number
}

func (e *substateEnv) SetNumber(number uint64) {
	e.Number = number
}

func (e *substateEnv) GetTimestamp() uint64 {
	return e.Timestamp
}

func (e *substateEnv) SetTimestamp(timestamp uint64) {
	e.Timestamp = timestamp
}

func (e *substateEnv) GetBaseFee() *big.Int {
	return e.BaseFee
}

func (e *substateEnv) SetBaseFee(baseFee *big.Int) {
	e.BaseFee = baseFee
}

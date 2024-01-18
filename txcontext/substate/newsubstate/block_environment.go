package newsubstate

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
)

func NewBlockEnvironment(env *substate.Env) txcontext.BlockEnvironment {
	return &blockEnvironment{env}
}

type blockEnvironment struct {
	*substate.Env
}

func (e *blockEnvironment) GetBlockHash(block uint64) common.Hash {
	return common.Hash(e.BlockHashes[block])
}

func (e *blockEnvironment) GetCoinbase() common.Address {
	return common.Address(e.Coinbase)
}

func (e *blockEnvironment) GetDifficulty() *big.Int {
	return e.Difficulty
}

func (e *blockEnvironment) GetGasLimit() uint64 {
	return e.GasLimit
}

func (e *blockEnvironment) GetNumber() uint64 {
	return e.Number
}

func (e *blockEnvironment) GetTimestamp() uint64 {
	return e.Timestamp
}

func (e *blockEnvironment) GetBaseFee() *big.Int {
	return e.BaseFee
}

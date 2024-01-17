package substate_transaction

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/executor/transaction"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use NewSubstateEnv instead.
func NewOldSubstateEnv(env *substate.SubstateEnv) transaction.BlockEnvironment {
	return &oldSubstateEnv{env}
}

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use substateEnv instead.
type oldSubstateEnv struct {
	*substate.SubstateEnv
}

func (e *oldSubstateEnv) GetBlockHash(block uint64) common.Hash {
	return e.BlockHashes[block]
}

func (e *oldSubstateEnv) SetBlockHash(block uint64, hash common.Hash) {
	e.BlockHashes[block] = hash
}

func (e *oldSubstateEnv) GetCoinbase() common.Address {
	return e.Coinbase
}

func (e *oldSubstateEnv) SetCoinbase(coinbase common.Address) {
	e.Coinbase = coinbase
}

func (e *oldSubstateEnv) GetDifficulty() *big.Int {
	return e.Difficulty
}

func (e *oldSubstateEnv) SetDifficulty(difficulty *big.Int) {
	e.Difficulty = difficulty
}

func (e *oldSubstateEnv) GetGasLimit() uint64 {
	return e.GasLimit
}

func (e *oldSubstateEnv) SetGasLimit(gasLimit uint64) {
	e.GasLimit = gasLimit
}

func (e *oldSubstateEnv) GetNumber() uint64 {
	return e.Number
}

func (e *oldSubstateEnv) SetNumber(number uint64) {
	e.Number = number
}

func (e *oldSubstateEnv) GetTimestamp() uint64 {
	return e.Timestamp
}

func (e *oldSubstateEnv) SetTimestamp(timestamp uint64) {
	e.Timestamp = timestamp
}

func (e *oldSubstateEnv) GetBaseFee() *big.Int {
	return e.BaseFee
}

func (e *oldSubstateEnv) SetBaseFee(baseFee *big.Int) {
	e.BaseFee = baseFee
}

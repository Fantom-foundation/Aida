package substate

import (
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Aida/txcontext"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use NewSubstateEnv instead.
func NewBlockEnvironment(env *substate.SubstateEnv) txcontext.BlockEnvironment {
	return &blockEnvironment{env}
}

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use substateEnv instead.
type blockEnvironment struct {
	*substate.SubstateEnv
}

func (e *blockEnvironment) GetBlockHash(block uint64) (common.Hash, error) {
	if e.BlockHashes == nil {
		return common.Hash{}, fmt.Errorf("getHash(%d) invoked, no blockhashes provided", block)
	}
	h, ok := e.BlockHashes[block]
	if !ok {
		return common.Hash(h), fmt.Errorf("getHash(%d) invoked, blockhash for that block not provided", block)
	}
	return common.Hash(h), nil
}

func (e *blockEnvironment) GetCoinbase() common.Address {
	return e.Coinbase
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

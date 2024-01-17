package substate_transaction

import (
	"github.com/Fantom-foundation/Aida/executor/transaction"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/types"
)

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use NewSubstateData instead.
func NewOldSubstateData(data *substate.Substate) SubstateData {
	return &oldSubstateData{data}
}

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use substateData instead.
type oldSubstateData struct {
	*substate.Substate
}

func (t *oldSubstateData) GetInputAlloc() transaction.WorldState {
	return NewOldSubstateAlloc(t.InputAlloc)
}

func (t *oldSubstateData) GetOutputAlloc() transaction.WorldState {
	return NewOldSubstateAlloc(t.OutputAlloc)
}

func (t *oldSubstateData) GetEnv() transaction.BlockEnvironment {
	return NewOldSubstateEnv(t.Env)
}

func (t *oldSubstateData) GetMessage() types.Message {
	return t.Message.AsMessage()
}

func (t *oldSubstateData) GetResult() transaction.Receipt {
	return NewOldSubstateResult(t.Result)
}

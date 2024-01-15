package transaction

import (
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/types"
)

func NewOldSubstateData(data *substate.Substate) SubstateData {
	return &oldSubstateData{data}
}

type oldSubstateData struct {
	*substate.Substate
}

func (t *oldSubstateData) GetInputAlloc() Alloc {
	return NewOldSubstateAlloc(t.InputAlloc)
}

func (t *oldSubstateData) GetOutputAlloc() Alloc {
	return NewOldSubstateAlloc(t.OutputAlloc)
}

func (t *oldSubstateData) GetEnv() Env {
	return NewOldSubstateEnv(t.Env)
}

func (t *oldSubstateData) GetMessage() types.Message {
	return t.Message.AsMessage()
}

func (t *oldSubstateData) GetResult() Result {
	return NewOldSubstateResult(t.Result)
}

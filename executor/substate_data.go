package executor

import (
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/types"
)

func NewSubstateData(data *substate.Substate) TransactionData {
	return &substateData{data}
}

type substateData struct {
	data *substate.Substate
}

func (t *substateData) GetInputAlloc() substate.SubstateAlloc {
	return t.data.InputAlloc
}

func (t *substateData) GetOutputAlloc() substate.SubstateAlloc {
	return t.data.OutputAlloc
}

func (t *substateData) GetEnv() *substate.SubstateEnv {
	return t.data.Env
}

func (t *substateData) GetMessage() types.Message {
	return t.data.Message.AsMessage()
}

func (t *substateData) GetResult() *substate.SubstateResult {
	return t.data.Result
}

package substate

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core"
)

func NewTxContext(data *substate.Substate) txcontext.TxContext {
	return &substateData{data}
}

type substateData struct {
	*substate.Substate
}

func (t *substateData) GetInputState() txcontext.WorldState {
	return NewWorldState(t.InputAlloc)
}

func (t *substateData) GetOutputState() txcontext.WorldState {
	return NewWorldState(t.OutputAlloc)
}

func (t *substateData) GetBlockEnvironment() txcontext.BlockEnvironment {
	return NewBlockEnvironment(t.Env)
}

func (t *substateData) GetMessage() core.Message {
	return t.Message.AsMessage()
}

func (t *substateData) GetReceipt() txcontext.Receipt {
	return NewReceipt(t.Result)
}

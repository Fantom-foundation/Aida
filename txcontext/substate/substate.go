package substate

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/types"
)

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use Newtransaction instead.
func NewTxContextWithValidation(data *substate.Substate) txcontext.WithValidation {
	return &substateData{data}
}

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use substateData instead.
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

func (t *substateData) GetMessage() types.Message {
	return t.Message.AsMessage()
}

func (t *substateData) GetReceipt() txcontext.Receipt {
	return NewReceipt(t.Result)
}

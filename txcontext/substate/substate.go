package substate

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

func NewTxContext(data *substate.Substate) txcontext.TxContext {
	return &substateData{data}
}

type substateData struct {
	*substate.Substate
}

func (t *substateData) GetResult() txcontext.Result {
	return NewResult(t.Result)
}

func (t *substateData) GetStateHash() common.Hash {
	// ignored
	return common.Hash{}
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

type substateResult struct {
}

func (s substateResult) GetReceipt() txcontext.Receipt {
	//TODO implement me
	panic("implement me")
}

func (s substateResult) GetRawResult() ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (s substateResult) GetGasUsed() uint64 {
	//TODO implement me
	panic("implement me")
}

package newsubstate

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
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
	var list types.AccessList
	for _, tuple := range t.Message.AccessList {
		var keys []common.Hash
		for _, key := range tuple.StorageKeys {
			keys = append(keys, common.Hash(key))
		}
		list = append(list, types.AccessTuple{Address: common.Address(tuple.Address), StorageKeys: keys})
	}
	return types.NewMessage(common.Address(t.Message.From), (*common.Address)(t.Message.To), t.Message.Nonce, t.Message.Value, t.Message.Gas, t.Message.GasPrice, t.Message.GasFeeCap, t.Message.GasTipCap, t.Message.Data, list, !t.Message.CheckNonce)
}

func (t *substateData) GetReceipt() txcontext.Receipt {
	return NewReceipt(t.Result)
}

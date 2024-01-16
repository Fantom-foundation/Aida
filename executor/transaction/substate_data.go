package transaction

import (
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type SubstateData interface {
	InputValidationData
	ExecutionData
	OutputValidationData
}

func NewSubstateData(data *substate.Substate) SubstateData {
	return &substateData{data}
}

type substateData struct {
	*substate.Substate
}

func (t *substateData) GetInputAlloc() WorldState {
	return NewSubstateAlloc(t.InputAlloc)
}

func (t *substateData) GetOutputAlloc() WorldState {
	return NewSubstateAlloc(t.OutputAlloc)
}

func (t *substateData) GetEnv() BlockEnvironment {
	return NewSubstateEnv(t.Env)
}

func (t *substateData) GetMessage() types.Message {
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

func (t *substateData) GetResult() TransactionReceipt {
	return NewSubstateResult(t.Result)
}

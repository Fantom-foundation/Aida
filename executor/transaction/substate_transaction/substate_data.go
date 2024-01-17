package substate_transaction

import (
	"github.com/Fantom-foundation/Aida/executor/transaction"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type SubstateData interface {
	transaction.InputValidationData
	transaction.ExecutionData
	transaction.OutputValidationData
}

func NewSubstateData(data *substate.Substate) SubstateData {
	return &substateData{data}
}

type substateData struct {
	*substate.Substate
}

func (t *substateData) GetInputAlloc() transaction.WorldState {
	return NewSubstateAlloc(t.InputAlloc)
}

func (t *substateData) GetOutputAlloc() transaction.WorldState {
	return NewSubstateAlloc(t.OutputAlloc)
}

func (t *substateData) GetEnv() transaction.BlockEnvironment {
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

func (t *substateData) GetResult() transaction.Receipt {
	return NewSubstateResult(t.Result)
}

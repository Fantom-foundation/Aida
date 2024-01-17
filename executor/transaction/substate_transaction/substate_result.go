package substate_transaction

import (
	"github.com/Fantom-foundation/Aida/executor/transaction"
	substateCommon "github.com/Fantom-foundation/Substate/geth/common"
	substateTypes "github.com/Fantom-foundation/Substate/geth/types"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// todo logs

func NewSubstateResult(res *substate.Result) transaction.Receipt {
	return &substateResult{res}
}

type substateResult struct {
	*substate.Result
}

func (r *substateResult) GetStatus() uint64 {
	return r.Status
}

func (r *substateResult) SetStatus(status uint64) {
	r.Status = status
}

func (r *substateResult) GetBloom() types.Bloom {
	return types.Bloom(r.Bloom)
}

func (r *substateResult) SetBloom(bloom types.Bloom) {
	r.Bloom = substateTypes.Bloom(bloom)
}

func (r *substateResult) GetLogs() []*types.Log {
	panic("how to return without iterating")
}

func (r *substateResult) SetLogs(logs []*types.Log) {
	panic("how to set without iterating")
}

func (r *substateResult) GetContractAddress() common.Address {
	return common.Address(r.ContractAddress)
}

func (r *substateResult) SetContractAddress(contractAddress common.Address) {
	r.ContractAddress = substateCommon.Address(contractAddress)
}

func (r *substateResult) GetGasUsed() uint64 {
	return r.GasUsed
}

func (r *substateResult) SetGasUsed(gasUsed uint64) {
	r.GasUsed = gasUsed
}

func (r *substateResult) Equal(y transaction.Receipt) bool {
	return transaction.ReceiptEqual(r, y)
}

package transaction

import (
	substateCommon "github.com/Fantom-foundation/Substate/geth/common"
	substateTypes "github.com/Fantom-foundation/Substate/geth/types"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// todo logs

func NewSubstateResult(res *substate.Result) Result {
	return &substateResult{res: res}
}

type substateResult struct {
	resultEqual
	res *substate.Result
}

func (r *substateResult) GetStatus() uint64 {
	return r.res.Status
}

func (r *substateResult) SetStatus(status uint64) {
	r.res.Status = status
}

func (r *substateResult) GetBloom() types.Bloom {
	return types.Bloom(r.res.Bloom)
}

func (r *substateResult) SetBloom(bloom types.Bloom) {
	r.res.Bloom = substateTypes.Bloom(bloom)
}

func (r *substateResult) GetLogs() []*types.Log {
	panic("how to return without iterating") // todo mby some transformations like bloom
}

func (r *substateResult) SetLogs(logs []*types.Log) {
	panic("how to set without iterating") // todo mby some transformations like bloom
}

func (r *substateResult) GetContractAddress() common.Address {
	return common.Address(r.res.ContractAddress)
}

func (r *substateResult) SetContractAddress(contractAddress common.Address) {
	r.res.ContractAddress = substateCommon.Address(contractAddress)
}

func (r *substateResult) GetGasUsed() uint64 {
	return r.res.GasUsed
}

func (r *substateResult) SetGasUsed(gasUsed uint64) {
	r.res.GasUsed = gasUsed
}

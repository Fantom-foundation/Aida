package transaction

import (
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// todo logs

func NewOldSubstateResult(res *substate.SubstateResult) Result {
	return &oldSubstateResult{res}
}

type oldSubstateResult struct {
	*substate.SubstateResult
}

func (r *oldSubstateResult) GetStatus() uint64 {
	return r.Status
}

func (r *oldSubstateResult) SetStatus(status uint64) {
	r.Status = status
}

func (r *oldSubstateResult) GetBloom() types.Bloom {
	return r.Bloom
}

func (r *oldSubstateResult) SetBloom(bloom types.Bloom) {
	r.Bloom = bloom
}

func (r *oldSubstateResult) GetLogs() []*types.Log {
	return r.Logs
}

func (r *oldSubstateResult) SetLogs(logs []*types.Log) {
	r.Logs = logs
}

func (r *oldSubstateResult) GetContractAddress() common.Address {
	return r.ContractAddress
}

func (r *oldSubstateResult) SetContractAddress(contractAddress common.Address) {
	r.ContractAddress = contractAddress
}

func (r *oldSubstateResult) GetGasUsed() uint64 {
	return r.GasUsed
}

func (r *oldSubstateResult) SetGasUsed(gasUsed uint64) {
	r.GasUsed = gasUsed
}

func (r *oldSubstateResult) Equal(y Result) bool {
	return resultEqual(r, y)
}

package newsubstate

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// todo logs

func NewReceipt(res *substate.Result) *result {
	return &result{res}
}

type result struct {
	*substate.Result
}

func (r *result) GetReceipt() txcontext.Receipt {
	return r
}

func (r *result) GetRawResult() ([]byte, error) {
	return nil, nil
}

func (r *result) GetStatus() uint64 {
	return r.Status
}

func (r *result) GetBloom() types.Bloom {
	return types.Bloom(r.Bloom)
}

func (r *result) GetLogs() []*types.Log {
	panic("how to return without iterating")
}

func (r *result) GetContractAddress() common.Address {
	return common.Address(r.ContractAddress)
}

func (r *result) GetGasUsed() uint64 {
	return r.GasUsed
}

func (r *result) Equal(y txcontext.Receipt) bool {
	return txcontext.ReceiptEqual(r, y)
}

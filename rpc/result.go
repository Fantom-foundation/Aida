package rpc

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func NewStatusSuccessfulResult(gasUsed uint64, res []byte) txcontext.Receipt {
	return &result{
		status:  types.ReceiptStatusSuccessful,
		gasUsed: gasUsed,
		result:  res,
	}
}

func NewStatusFailedResult(gasUsed uint64, res []byte) txcontext.Receipt {
	return &result{
		status:  types.ReceiptStatusFailed,
		gasUsed: gasUsed,
		result:  res,
	}
}

func NewErrorResult(gasUsed uint64, err error) txcontext.Receipt {
	return &result{
		status:  types.ReceiptStatusFailed,
		gasUsed: gasUsed,
		err:     err,
	}
}

type result struct {
	status  uint64
	gasUsed uint64
	result  []byte
	err     error
}

func (r *result) GetStatus() uint64 {
	return r.status
}

func (r *result) GetBloom() types.Bloom {
	return types.Bloom{}
}

func (r *result) GetLogs() []*types.Log {
	return []*types.Log{}
}

func (r *result) GetContractAddress() common.Address {
	return common.Address{}
}

func (r *result) GetGasUsed() uint64 {
	return r.gasUsed
}

func (r *result) Equal(y txcontext.Receipt) bool {
	//TODO implement me
	panic("implement me")
}

func (r *result) GetResult() txcontext.EvmResult {
	return txcontext.EvmResult{
		Message: r.result,
		Err:     r.err,
	}
}

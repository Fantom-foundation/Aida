package transaction

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func NewVmResult(status uint64, bloom types.Bloom, logs []*types.Log, contractAddress common.Address, gasUsed uint64) TransactionReceipt {
	return &vmResult{status: status, bloom: bloom, logs: logs, contractAddress: contractAddress, gasUsed: gasUsed}
}

type vmResult struct {
	status          uint64
	bloom           types.Bloom
	logs            []*types.Log
	contractAddress common.Address
	gasUsed         uint64
}

func (r *vmResult) GetStatus() uint64 {
	return r.status
}

func (r *vmResult) SetStatus(status uint64) {
	r.status = status
}

func (r *vmResult) GetBloom() types.Bloom {
	return r.bloom
}

func (r *vmResult) SetBloom(bloom types.Bloom) {
	r.bloom = bloom
}

func (r *vmResult) GetLogs() []*types.Log {
	return r.logs
}

func (r *vmResult) SetLogs(logs []*types.Log) {
	r.logs = logs
}

func (r *vmResult) GetContractAddress() common.Address {
	return r.contractAddress
}

func (r *vmResult) SetContractAddress(contractAddress common.Address) {
	r.contractAddress = contractAddress
}

func (r *vmResult) GetGasUsed() uint64 {
	return r.gasUsed
}

func (r *vmResult) SetGasUsed(gasUsed uint64) {
	r.gasUsed = gasUsed
}

func (r *vmResult) Equal(y TransactionReceipt) bool {
	return transactionReceiptEqual(r, y)
}

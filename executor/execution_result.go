package executor

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type transactionResult struct {
	result          []byte
	err             error
	status          uint64
	bloom           types.Bloom
	logs            []*types.Log
	contractAddress common.Address
	gasUsed         uint64
}

func (r transactionResult) GetReceipt() txcontext.Receipt {
	// transactionResult implements both txcontext.Result and txcontext.Receipt
	return r
}

func (r transactionResult) GetRawResult() ([]byte, error) {
	return r.result, r.err
}

func (r transactionResult) GetGasUsed() uint64 {
	return r.gasUsed
}

func (r transactionResult) GetStatus() uint64 {
	return r.status
}

func (r transactionResult) GetBloom() types.Bloom {
	return r.bloom
}

func (r transactionResult) GetLogs() []*types.Log {
	return r.logs
}

func (r transactionResult) GetContractAddress() common.Address {
	return r.contractAddress
}

func (r transactionResult) Equal(y txcontext.Receipt) bool {
	return txcontext.ReceiptEqual(r, y)
}

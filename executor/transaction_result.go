package executor

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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

func (r transactionResult) String() string {
	return fmt.Sprintf("Status: %v\nBloom: %s\nContract Address: %s\nGas Used: %v\nLogs: %v\n", r.status, string(r.bloom.Bytes()), r.contractAddress, r.gasUsed, r.logs)
}

func newTransactionResult(logs []*types.Log, msg core.Message, msgResult *evmcore.ExecutionResult, err error, origin common.Address) transactionResult {
	var (
		contract common.Address
		gasUsed  uint64
		status   uint64
	)

	if to := msg.To(); to == nil {
		contract = crypto.CreateAddress(origin, msg.Nonce())
	}

	var returnData []byte
	if msgResult != nil {
		returnData = msgResult.Return()
		gasUsed = msgResult.UsedGas
		if msgResult.Failed() {
			status = types.ReceiptStatusFailed
		} else {
			status = types.ReceiptStatusSuccessful
		}
	}

	return transactionResult{
		result:          returnData,
		err:             err,
		contractAddress: contract,
		logs:            logs,
		bloom:           types.BytesToBloom(types.LogsBloom(logs)),
		status:          status,
		gasUsed:         gasUsed,
	}
}

func newPseudoExecutionResult() txcontext.Result {
	return transactionResult{
		result:          []byte{},
		err:             nil,
		status:          types.ReceiptStatusSuccessful,
		bloom:           types.Bloom{},
		logs:            nil,
		contractAddress: common.Address{},
		gasUsed:         0,
	}
}

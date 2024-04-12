package substate

import (
	"fmt"

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
	// todo remove iteration once fantom types are created
	logs := make([]*types.Log, 0)
	for _, l := range r.Logs {
		topics := make([]common.Hash, 0)
		for _, t := range l.Topics {
			topics = append(topics, common.Hash(t))
		}

		logs = append(logs, &types.Log{
			Address:     common.Address(l.Address),
			Topics:      topics,
			Data:        l.Data,
			BlockNumber: l.BlockNumber,
			TxHash:      common.Hash(l.TxHash),
			TxIndex:     l.TxIndex,
			BlockHash:   common.Hash(l.BlockHash),
			Index:       l.Index,
			Removed:     l.Removed,
		})
	}

	return logs
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

func (r *result) String() string {
	return fmt.Sprintf("Status: %v\nBloom: %s\nContract Address: %s\nGas Used: %v\nLogs: %v\n", r.Status, string(r.Bloom.Bytes()), r.ContractAddress, r.GasUsed, r.Logs)
}

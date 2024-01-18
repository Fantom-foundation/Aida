package newsubstate

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// todo logs

func NewReceipt(res *substate.Result) txcontext.Receipt {
	return &receipt{res}
}

type receipt struct {
	*substate.Result
}

func (r *receipt) GetStatus() uint64 {
	return r.Status
}

func (r *receipt) GetBloom() types.Bloom {
	return types.Bloom(r.Bloom)
}

func (r *receipt) GetLogs() []*types.Log {
	panic("how to return without iterating")
}

func (r *receipt) GetContractAddress() common.Address {
	return common.Address(r.ContractAddress)
}

func (r *receipt) GetGasUsed() uint64 {
	return r.GasUsed
}

func (r *receipt) Equal(y txcontext.Receipt) bool {
	return txcontext.ReceiptEqual(r, y)
}

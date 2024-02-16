package substate

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// todo logs

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use NewSubstateResult instead.
func NewReceipt(res *substate.SubstateResult) txcontext.Receipt {
	return &receipt{res}
}

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use substateResult instead.
type receipt struct {
	*substate.SubstateResult
}

func (r *receipt) GetStatus() uint64 {
	return r.Status
}
func (r *receipt) GetBloom() types.Bloom {
	return r.Bloom
}

func (r *receipt) GetLogs() []*types.Log {
	return r.Logs
}

func (r *receipt) GetContractAddress() common.Address {
	return r.ContractAddress
}

func (r *receipt) GetGasUsed() uint64 {
	return r.GasUsed
}

func (r *receipt) GetResult() txcontext.EvmResult {
	return txcontext.EvmResult{}
}

func (r *receipt) Equal(y txcontext.Receipt) bool {
	return txcontext.ReceiptEqual(r, y)
}

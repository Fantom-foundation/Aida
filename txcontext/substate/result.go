package substate

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/txcontext"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// todo logs

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use NewSubstateResult instead.
func NewResult(res *substate.SubstateResult) *result {
	return &result{res}
}

// Deprecated: This is a workaround before oldSubstate repository is migrated to new structure.
// Use substateResult instead.
type result struct {
	*substate.SubstateResult
}

func (r *result) GetRawResult() ([]byte, error) {
	// we do not have access to this in substate
	return nil, nil
}

func (r *result) GetReceipt() txcontext.Receipt {
	// result implements both txcontext.Result and txcontext.Receipt
	return r
}

func (r *result) GetStatus() uint64 {
	return r.Status
}
func (r *result) GetBloom() types.Bloom {
	return r.Bloom
}

func (r *result) GetLogs() []*types.Log {
	return r.Logs
}

func (r *result) GetContractAddress() common.Address {
	return r.ContractAddress
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

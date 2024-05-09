// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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

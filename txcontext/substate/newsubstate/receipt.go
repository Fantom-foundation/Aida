// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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

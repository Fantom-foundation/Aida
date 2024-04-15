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

package substate

import (
	"testing"

	"github.com/Fantom-foundation/Aida/txcontext"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestReceipt_EqualStatus tests whether Equal works with status.
func TestReceipt_EqualStatus(t *testing.T) {
	res := &substate.SubstateResult{Status: 0}
	comparedRes := &substate.SubstateResult{Status: 1}

	if txcontext.ReceiptEqual(NewResult(res), NewResult(comparedRes)) {
		t.Fatal("results status are different but equal returned true")
	}

	comparedRes.Status = res.Status
	if !txcontext.ReceiptEqual(NewResult(res), NewResult(comparedRes)) {
		t.Fatal("results status are same but equal returned false")
	}
}

// TestReceipt_EqualBloom tests whether Equal works with bloom.
func TestReceipt_EqualBloom(t *testing.T) {
	res := &substate.SubstateResult{Bloom: types.Bloom{0}}
	comparedRes := &substate.SubstateResult{Bloom: types.Bloom{1}}

	if txcontext.ReceiptEqual(NewResult(res), NewResult(comparedRes)) {
		t.Fatal("results Bloom are different but equal returned true")
	}

	comparedRes.Bloom = res.Bloom
	if !txcontext.ReceiptEqual(NewResult(res), NewResult(comparedRes)) {
		t.Fatal("results Bloom are same but equal returned false")
	}
}

// TestReceipt_EqualLogs tests whether Equal works with logs.
func TestReceipt_EqualLogs(t *testing.T) {
	res := &substate.SubstateResult{Logs: []*types.Log{{Address: common.Address{0}}}}
	comparedRes := &substate.SubstateResult{Logs: []*types.Log{{Address: common.Address{1}}}}

	if txcontext.ReceiptEqual(NewResult(res), NewResult(comparedRes)) {
		t.Fatal("results Log are different but equal returned true")
	}

	comparedRes.Logs = res.Logs
	if !txcontext.ReceiptEqual(NewResult(res), NewResult(comparedRes)) {
		t.Fatal("results Log are same but equal returned false")
	}
}

// TestReceipt_EqualContractAddress tests whether Equal works with contract address.
func TestReceipt_EqualContractAddress(t *testing.T) {
	res := &substate.SubstateResult{ContractAddress: common.Address{0}}
	comparedRes := &substate.SubstateResult{ContractAddress: common.Address{1}}

	if txcontext.ReceiptEqual(NewResult(res), NewResult(comparedRes)) {
		t.Fatal("results ContractAddress are different but equal returned true")
	}

	comparedRes.ContractAddress = res.ContractAddress
	if !txcontext.ReceiptEqual(NewResult(res), NewResult(comparedRes)) {
		t.Fatal("results ContractAddress are same but equal returned false")
	}
}

// TestReceipt_EqualGasUsed tests whether Equal works with contract has correct format.
func TestReceipt_EqualGasUsed(t *testing.T) {
	res := &substate.SubstateResult{GasUsed: 0}
	comparedRes := &substate.SubstateResult{GasUsed: 1}

	if txcontext.ReceiptEqual(NewResult(res), NewResult(comparedRes)) {
		t.Fatal("results GasUsed are different but equal returned true")
	}

	comparedRes.GasUsed = res.GasUsed
	if !txcontext.ReceiptEqual(NewResult(res), NewResult(comparedRes)) {
		t.Fatal("results GasUsed are same but equal returned false")
	}
}

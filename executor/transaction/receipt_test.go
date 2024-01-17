package transaction

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor/transaction/substate_transaction"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestReceipt_EqualStatus(t *testing.T) {
	res := &substate.SubstateResult{Status: 0}
	comparedRes := &substate.SubstateResult{Status: 1}

	if ReceiptEqual(substate_transaction.NewOldSubstateResult(res), substate_transaction.NewOldSubstateResult(comparedRes)) {
		t.Fatal("results status are different but equal returned true")
	}

	comparedRes.Status = res.Status
	if !ReceiptEqual(substate_transaction.NewOldSubstateResult(res), substate_transaction.NewOldSubstateResult(comparedRes)) {
		t.Fatal("results status are same but equal returned false")
	}
}

func TestReceipt_EqualBloom(t *testing.T) {
	res := &substate.SubstateResult{Bloom: types.Bloom{0}}
	comparedRes := &substate.SubstateResult{Bloom: types.Bloom{1}}

	if ReceiptEqual(substate_transaction.NewOldSubstateResult(res), substate_transaction.NewOldSubstateResult(comparedRes)) {
		t.Fatal("results Bloom are different but equal returned true")
	}

	comparedRes.Bloom = res.Bloom
	if !ReceiptEqual(substate_transaction.NewOldSubstateResult(res), substate_transaction.NewOldSubstateResult(comparedRes)) {
		t.Fatal("results Bloom are same but equal returned false")
	}
}

func TestReceipt_EqualLogs(t *testing.T) {
	res := &substate.SubstateResult{Logs: []*types.Log{{Address: common.Address{0}}}}
	comparedRes := &substate.SubstateResult{Logs: []*types.Log{{Address: common.Address{1}}}}

	if ReceiptEqual(substate_transaction.NewOldSubstateResult(res), substate_transaction.NewOldSubstateResult(comparedRes)) {
		t.Fatal("results Log are different but equal returned true")
	}

	comparedRes.Logs = res.Logs
	if !ReceiptEqual(substate_transaction.NewOldSubstateResult(res), substate_transaction.NewOldSubstateResult(comparedRes)) {
		t.Fatal("results Log are same but equal returned false")
	}
}

func TestReceipt_EqualContractAddress(t *testing.T) {
	res := &substate.SubstateResult{ContractAddress: common.Address{0}}
	comparedRes := &substate.SubstateResult{ContractAddress: common.Address{1}}

	if ReceiptEqual(substate_transaction.NewOldSubstateResult(res), substate_transaction.NewOldSubstateResult(comparedRes)) {
		t.Fatal("results ContractAddress are different but equal returned true")
	}

	comparedRes.ContractAddress = res.ContractAddress
	if !ReceiptEqual(substate_transaction.NewOldSubstateResult(res), substate_transaction.NewOldSubstateResult(comparedRes)) {
		t.Fatal("results ContractAddress are same but equal returned false")
	}
}

func TestReceipt_EqualGasUsed(t *testing.T) {
	res := &substate.SubstateResult{GasUsed: 0}
	comparedRes := &substate.SubstateResult{GasUsed: 1}

	if ReceiptEqual(substate_transaction.NewOldSubstateResult(res), substate_transaction.NewOldSubstateResult(comparedRes)) {
		t.Fatal("results GasUsed are different but equal returned true")
	}

	comparedRes.GasUsed = res.GasUsed
	if !ReceiptEqual(substate_transaction.NewOldSubstateResult(res), substate_transaction.NewOldSubstateResult(comparedRes)) {
		t.Fatal("results GasUsed are same but equal returned false")
	}
}

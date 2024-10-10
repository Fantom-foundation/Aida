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

package executor

import (
	"math/big"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"go.uber.org/mock/gomock"
)

// TestPrepareBlockCtx tests a creation of block context from substate environment.
func TestPrepareBlockCtx(t *testing.T) {
	gaslimit := uint64(10000000)
	blocknum := uint64(4600000)
	basefee := big.NewInt(12345)
	env := substatecontext.NewBlockEnvironment(&substate.Env{Difficulty: big.NewInt(1), GasLimit: gaslimit, Number: blocknum, Timestamp: 1675961395, BaseFee: basefee})

	var hashError error
	// BlockHashes are nil, expect an error
	blockCtx := prepareBlockCtx(env, &hashError)

	if blocknum != blockCtx.BlockNumber.Uint64() {
		t.Fatalf("Wrong block number")
	}
	if gaslimit != blockCtx.GasLimit {
		t.Fatalf("Wrong amount of gas limit")
	}
	if basefee.Cmp(blockCtx.BaseFee) != 0 {
		t.Fatalf("Wrong base fee")
	}
	if hashError != nil {
		t.Fatalf("Hash error; %v", hashError)
	}
}

func TestMakeTxProcessor_CanSelectBetweenProcessorImplementations(t *testing.T) {
	isOpera := func(t *testing.T, p processor, name string) {
		processor, ok := p.(*aidaProcessor)
		if !ok {
			t.Fatalf("Expected aidaProcessor from '%s', got %T", name, p)
		}

		cfg := processor.vmCfg
		if !cfg.ChargeExcessGas ||
			!cfg.IgnoreGasFeeCap ||
			!cfg.InsufficientBalanceIsNotAnError ||
			!cfg.SkipTipPaymentToCoinbase {
			t.Fatalf("Expected Opera features to be enabled")
		}
	}
	isEthereum := func(t *testing.T, p processor, name string) {
		processor, ok := p.(*aidaProcessor)
		if !ok {
			t.Fatalf("Expected aidaProcessor from '%s', got %T", name, p)
		}

		cfg := processor.vmCfg
		if cfg.ChargeExcessGas ||
			cfg.IgnoreGasFeeCap ||
			cfg.InsufficientBalanceIsNotAnError ||
			cfg.SkipTipPaymentToCoinbase {
			t.Fatalf("Expected Opera features to be disabled")
		}
	}
	isTosca := func(t *testing.T, p processor, name string) {
		if _, ok := p.(*toscaProcessor); !ok {
			t.Fatalf("Expected toscaProcessor from '%s', got %T", name, p)
		}
	}

	tests := map[string]func(*testing.T, processor, string){
		"":         isOpera,
		"opera":    isOpera,
		"ethereum": isEthereum,
	}

	for name := range tosca.GetAllRegisteredProcessorFactories() {
		if _, present := tests[name]; !present {
			tests[name] = isTosca
		}
	}

	for name, check := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := &utils.Config{
				ChainID: utils.MainnetChainID,
				EvmImpl: name,
				VmImpl:  "geth",
			}
			p, err := MakeTxProcessor(cfg)
			if err != nil {
				t.Fatalf("Failed to create tx processor; %v", err)
			}
			check(t, p.processor, name)
		})
	}

}

/*
func TestMakeAidaProcessor_CanChooseDifferentApplyMessage(t *testing.T) {
	cfg := utils.NewTestConfig(t, 250, 0, 1, false, "")
	tests := []struct {
		name    string
		evmImpl string
		want    applyMessage
	}{
		{
			name:    "expect_applyMessageUsingGeth",
			evmImpl: "aida-geth",
			want:    applyMessageUsingGeth,
		},
		{
			name:    "expect_applyMessageUsingSonic",
			evmImpl: "aida",
			want:    applyMessageUsingSonic,
		},
		{
			name:    "expect_defaultsToApplyMessageUsingSonic",
			evmImpl: "",
			want:    applyMessageUsingSonic,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg.EvmImpl = test.evmImpl
			aidaProcessor := makeAidaProcessor(cfg, vm.Config{})

			if got, want := fmt.Sprintf("%p", aidaProcessor.applyMessage), fmt.Sprintf("%p", test.want); got != want {
				t.Errorf("unexpected apply func")
			}

		})
	}
}

func TestEthTestProcessor_DoesNotExecuteTransactionWhenBlobGasCouldExceed(t *testing.T) {
	p, err := MakeEthTestProcessor(&utils.Config{})
	if err != nil {
		t.Fatalf("cannot make eth test processor: %v", err)
	}
	ctrl := gomock.NewController(t)
	// Process is returned early - nothing is expected
	stateDb := state.NewMockStateDB(ctrl)

	ctx := &Context{State: stateDb}
	err = p.Process(State[txcontext.TxContext]{Data: ethtest.CreateTransactionThatFailsBlobGasExceedCheck(t)}, ctx)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	_, got := ctx.ExecutionResult.GetRawResult()
	want := "blob gas exceeds maximum"
	if !strings.EqualFold(got.Error(), want) {
		t.Errorf("unexpected error, got: %v, want: %v", got, want)
	}
}

func TestEthTestProcessor_DoesNotExecuteTransactionWithInvalidTxBytes(t *testing.T) {
	tests := []struct {
		name          string
		expectedError string
		data          txcontext.TxContext
	}{
		{
			name:          "fails_unmarshal",
			expectedError: "cannot unmarshal tx-bytes",
			data:          ethtest.CreateTransactionWithInvalidTxBytes(t),
		},
		{
			name:          "fails_validation",
			expectedError: "cannot validate sender",
			data:          ethtest.CreateTransactionThatFailsSenderValidation(t),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p, err := MakeEthTestProcessor(&utils.Config{ChainID: utils.EthTestsChainID})
			if err != nil {
				t.Fatalf("cannot make eth test processor: %v", err)
			}
			ctrl := gomock.NewController(t)
			// Process is returned early - no calls are expected
			stateDb := state.NewMockStateDB(ctrl)

			ctx := &Context{State: stateDb}
			err = p.Process(State[txcontext.TxContext]{Data: test.data}, ctx)
			if err != nil {
				t.Fatalf("run failed: %v", err)
			}

			_, got := ctx.ExecutionResult.GetRawResult()
			if !strings.Contains(got.Error(), test.expectedError) {
				t.Errorf("unexpected error, got: %v, want: %v", got, test.expectedError)
			}
		})
	}
}

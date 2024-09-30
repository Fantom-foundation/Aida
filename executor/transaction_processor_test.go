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
	"fmt"
	"math/big"
	"testing"

	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/ethereum/go-ethereum/core/vm"
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
	isAida := func(t *testing.T, p processor) {
		if _, ok := p.(*aidaProcessor); !ok {
			t.Fatalf("Expected aidaProcessor, got %T", p)
		}
	}
	isTosca := func(t *testing.T, p processor) {
		if _, ok := p.(*toscaProcessor); !ok {
			t.Fatalf("Expected toscaProcessor, got %T", p)
		}
	}

	tests := map[string]func(*testing.T, processor){
		"":     isAida,
		"aida": isAida,
		"Aida": isAida,
	}

	for name := range tosca.GetAllRegisteredProcessorFactories() {
		tests[name] = isTosca
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
			check(t, p.processor)
		})
	}

}

func TestMakeAidaProcessor_CanChooseDifferentApplyMessage(t *testing.T) {
	cfg := utils.NewTestConfig(t, 250, 0, 1, false, "")
	tests := []struct {
		name               string
		useGethTxProcessor bool
	}{
		{
			name:               "expect_applyMessageUsingGeth",
			useGethTxProcessor: true,
		},
		{
			name:               "expect_applyMessageUsingSonic",
			useGethTxProcessor: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg.UseGethTxProcessor = test.useGethTxProcessor
			aidaProcessor := makeAidaProcessor(cfg, vm.Config{})
			apply := aidaProcessor.applyMessageUsingSonic
			if cfg.UseGethTxProcessor {
				apply = aidaProcessor.applyMessageUsingGeth
			}

			if got, want := fmt.Sprintf("%p", aidaProcessor.applyMessage), fmt.Sprintf("%p", apply); got != want {
				t.Errorf("unexpected apply func")
			}

		})
	}
}

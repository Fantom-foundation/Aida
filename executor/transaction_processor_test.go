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

package executor

import (
	"math/big"
	"testing"

	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	substate "github.com/Fantom-foundation/Substate"
)

// TestPrepareBlockCtx tests a creation of block context from substate environment.
func TestPrepareBlockCtx(t *testing.T) {
	gaslimit := uint64(10000000)
	blocknum := uint64(4600000)
	basefee := big.NewInt(12345)
	env := substatecontext.NewBlockEnvironment(&substate.SubstateEnv{Difficulty: big.NewInt(1), GasLimit: gaslimit, Number: blocknum, Timestamp: 1675961395, BaseFee: basefee})

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

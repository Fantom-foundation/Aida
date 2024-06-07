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

package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/utils"
)

func TestTemporaryStatePrepper_DefaultDbImplementationIsOffTheChainStateDb(t *testing.T) {
	cfg := &utils.Config{}
	cfg.DbImpl = ""

	ext := MakeTemporaryStatePrepper(cfg)

	// check that temporaryOffTheChainStatePrepper is default
	if _, ok := ext.(*temporaryOffTheChainStatePrepper); !ok {
		t.Fatal("unexpected extension type")
	}
}

func TestTemporaryStatePrepper_OffTheChainDbImplementation(t *testing.T) {
	cfg := &utils.Config{}
	cfg.DbImpl = "off-the-chain"

	ext := MakeTemporaryStatePrepper(cfg)

	if _, ok := ext.(*temporaryOffTheChainStatePrepper); !ok {
		t.Fatal("unexpected extension type")
	}

}

func TestTemporaryStatePrepper_InMemoryDbImplementation(t *testing.T) {
	cfg := &utils.Config{}
	cfg.DbImpl = "in-memory"

	ext := MakeTemporaryStatePrepper(cfg)

	if _, ok := ext.(*temporaryInMemoryStatePrepper); !ok {
		t.Fatal("unexpected extension type")
	}
}

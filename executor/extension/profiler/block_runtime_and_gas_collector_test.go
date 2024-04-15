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

package profiler

import (
	"errors"
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

func TestBlockProfilerExtension_NoProfileIsCollectedIfDisabled(t *testing.T) {
	config := &utils.Config{}
	ext := MakeBlockRuntimeAndGasCollector(config)

	if _, ok := ext.(extension.NilExtension[txcontext.TxContext]); !ok {
		t.Errorf("profiler is enabled although not set in configuration")
	}
}

func TestBlockProfilerExtension_ProfileDbIsCreated(t *testing.T) {
	path := t.TempDir() + "/profile.db"
	config := &utils.Config{}
	config.ProfileBlocks = true
	config.ProfileDB = path

	ext := MakeBlockRuntimeAndGasCollector(config)

	if err := ext.PreRun(executor.State[txcontext.TxContext]{}, nil); err != nil {
		t.Fatalf("unexpected error during pre-run; %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Fatal("db was not created")
		}
		t.Fatalf("unexpected error; %v", err)
	}
}

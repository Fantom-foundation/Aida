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
	"fmt"
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/utils"
)

func TestCpuExtension_CollectsProfileDataIfEnabled(t *testing.T) {
	path := t.TempDir() + "/profile.dat"
	cfg := &utils.Config{}
	cfg.CPUProfile = path
	ext := MakeCpuProfiler[any](cfg)

	if err := ext.PreRun(executor.State[any]{}, nil); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}
	ext.PostRun(executor.State[any]{}, nil, nil)

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Errorf("no profile was collected")
	}
}

func TestCpuExtension_CollectsIntervalProfileDataIfEnabled(t *testing.T) {
	path := t.TempDir() + "/profile.dat"
	cfg := &utils.Config{}
	cfg.CPUProfile = path
	cfg.CPUProfilePerInterval = true
	ext := MakeCpuProfiler[any](cfg)

	if err := ext.PreRun(executor.State[any]{}, nil); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}

	for _, block := range []int{90_000, 120_000, 220_000} {
		if err := ext.PreBlock(executor.State[any]{Block: block}, nil); err != nil {
			t.Fatalf("failed to to run pre-block: %v", err)
		}
	}

	ext.PostRun(executor.State[any]{}, nil, nil)

	for _, interval := range []int{0, 1, 2} {
		file := fmt.Sprintf("%s_%05d", path, interval)
		if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
			t.Errorf("missing profile data file %v", file)
		}
	}
}

func TestCpuExtension_NpProfileIsCollectedIfDisabled(t *testing.T) {
	cfg := &utils.Config{}
	ext := MakeCpuProfiler[any](cfg)

	if _, ok := ext.(extension.NilExtension[any]); !ok {
		t.Errorf("profiler is enabled although not set in configuration")
	}
}

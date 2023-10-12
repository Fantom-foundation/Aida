package profiler_extensions

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
	config := &utils.Config{}
	config.CPUProfile = path
	ext := MakeCpuProfiler(config)

	if err := ext.PreRun(executor.State{}, nil); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}
	ext.PostRun(executor.State{}, nil, nil)

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Errorf("no profile was collected")
	}
}

func TestCpuExtension_CollectsIntervalProfileDataIfEnabled(t *testing.T) {
	path := t.TempDir() + "/profile.dat"
	config := &utils.Config{}
	config.CPUProfile = path
	config.CPUProfilePerInterval = true
	ext := MakeCpuProfiler(config)

	if err := ext.PreRun(executor.State{}, nil); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}

	for _, block := range []int{90_000, 120_000, 220_000} {
		if err := ext.PreBlock(executor.State{Block: block}, nil); err != nil {
			t.Fatalf("failed to to run pre-block: %v", err)
		}
	}

	ext.PostRun(executor.State{}, nil, nil)

	for _, interval := range []int{0, 1, 2} {
		file := fmt.Sprintf("%s_%05d", path, interval)
		if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
			t.Errorf("missing profile data file %v", file)
		}
	}
}

func TestCpuExtension_NpProfileIsCollectedIfDisabled(t *testing.T) {
	config := &utils.Config{}
	ext := MakeCpuProfiler(config)

	if _, ok := ext.(extension.NilExtension); !ok {
		t.Errorf("profiler is enabled although not set in configuration")
	}
}

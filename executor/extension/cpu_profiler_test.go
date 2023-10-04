package extension

import (
	"errors"
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
)

func TestCpuExtension_CollectsProfileDataIfEnabled(t *testing.T) {
	path := t.TempDir() + "/profile.dat"
	config := &utils.Config{}
	config.CPUProfile = path
	ext := MakeCpuProfiler[any](config)

	if err := ext.PreRun(executor.State[any]{}, nil); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}
	ext.PostRun(executor.State[any]{}, nil, nil)

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Errorf("no profile was collected")
	}
}

func TestCpuExtension_NpProfileIsCollectedIfDisabled(t *testing.T) {
	config := &utils.Config{}
	ext := MakeCpuProfiler[any](config)

	if _, ok := ext.(NilExtension[any]); !ok {
		t.Errorf("profiler is enabled although not set in configuration")
	}
}

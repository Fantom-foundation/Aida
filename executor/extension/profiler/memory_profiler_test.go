package profiler

import (
	"errors"
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/utils"
)

func TestMemoryProfiler_CollectsProfileDataIfEnabled(t *testing.T) {
	path := t.TempDir() + "/profile.dat"
	config := &utils.Config{}
	config.MemoryProfile = path
	ext := MakeMemoryProfiler[any](config)

	if err := ext.PreRun(executor.State[any]{}, nil); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}
	ext.PostRun(executor.State[any]{}, nil, nil)

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Errorf("no profile was collected")
	}
}

func TestMemoryProfiler_NoProfileIsCollectedIfDisabled(t *testing.T) {
	config := &utils.Config{}
	ext := MakeMemoryProfiler[any](config)

	if _, ok := ext.(extension.NilExtension[any]); !ok {
		t.Errorf("profiler is enabled although not set in configuration")
	}
}

package extension

import (
	"errors"
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
)

func TestMemoryProfiler_CollectsProfileDataIfEnabled(t *testing.T) {
	path := t.TempDir() + "/profile.dat"
	config := &utils.Config{}
	config.MemoryProfile = path
	ext := MakeMemoryProfiler(config)

	if err := ext.PreRun(executor.State{}, nil); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}
	ext.PostRun(executor.State{}, nil, nil)

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Errorf("no profile was collected")
	}
}

func TestMemoryProfiler_NoProfileIsCollectedIfDisabled(t *testing.T) {
	config := &utils.Config{}
	ext := MakeMemoryProfiler(config)

	if _, ok := ext.(NilExtension); !ok {
		t.Errorf("profiler is enabled although not set in configuration")
	}
}

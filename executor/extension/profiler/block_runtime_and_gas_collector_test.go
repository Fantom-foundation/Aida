package profiler

import (
	"errors"
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/utils"
)

func TestBlockProfilerExtension_NoProfileIsCollectedIfDisabled(t *testing.T) {
	config := &utils.Config{}
	ext := MakeBlockRuntimeAndGasCollector(config)

	if _, ok := ext.(extension.NilExtension); !ok {
		t.Errorf("profiler is enabled although not set in configuration")
	}
}

func TestBlockProfilerExtension_ProfileDbIsCreated(t *testing.T) {
	path := t.TempDir() + "/profile.db"
	config := &utils.Config{}
	config.ProfileBlocks = true
	config.ProfileDB = path

	ext := MakeBlockRuntimeAndGasCollector(config)

	if err := ext.PreRun(executor.State{}, nil); err != nil {
		t.Fatalf("unexpected error during pre-run; %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Fatal("db was not created")
		}
		t.Fatalf("unexpected error; %v", err)
	}
}

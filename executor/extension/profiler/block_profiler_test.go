package profiler

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/utils"
)

func TestBlockProfilerExtension_NoProfileIsCollectedIfDisabled(t *testing.T) {
	config := &utils.Config{}
	ext := MakeBlockProfiler(config)

	if _, ok := ext.(extension.NilExtension); !ok {
		t.Errorf("profiler is enabled although not set in configuration")
	}
}

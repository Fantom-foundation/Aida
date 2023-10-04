package extension

import (
	"testing"

	"github.com/Fantom-foundation/Aida/utils"
)

func TestAidaDbManager_NoManagerIsCreatedIfPathIsNotProvided(t *testing.T) {
	config := &utils.Config{}
	ext := MakeAidaDbManager(config)

	if _, ok := ext.(NilExtension); !ok {
		t.Errorf("profiler is enabled although not set in configuration")
	}
}

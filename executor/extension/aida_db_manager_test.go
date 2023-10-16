package extension

import (
	"testing"

	"github.com/Fantom-foundation/Aida/utils"
)

func TestAidaDbManager_NoManagerIsCreatedIfPathIsNotProvided(t *testing.T) {
	config := &utils.Config{}
	ext := MakeAidaDbManager[any](config)

	if _, ok := ext.(NilExtension[any]); !ok {
		t.Errorf("manager is enabled although not set in configuration")
	}
}

package extension

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestProgressLoggerExtension_NoPrimerIsCreatedIfDisabled(t *testing.T) {
	config := &utils.Config{}
	config.SkipPriming = true

	ext := MakeStateDbPrimer(config)
	if _, ok := ext.(NilExtension); !ok {
		t.Errorf("Primer is enabled although not set in configuration")
	}

}

func TestProgressLoggerExtension_PrimingDoesNotTriggerForExistingStateDb(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	config := &utils.Config{}
	config.SkipPriming = false
	config.StateDbSrc = "existing_state_db"

	log.EXPECT().Warning("Skipping priming")

	ext := &stateDbPrimer{
		config: config,
		log:    log,
	}

	ext.PreRun(executor.State{})

}

func TestProgressLoggerExtension_PrimingDoesTriggerForNonExistingStateDb(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	config := &utils.Config{}
	config.SkipPriming = false
	config.StateDbSrc = ""
	config.First = 2

	log.EXPECT().Noticef("Priming to block %v", config.First-1)

	ext := &stateDbPrimer{
		config: config,
		log:    log,
	}

	ext.PreRun(executor.State{})
}

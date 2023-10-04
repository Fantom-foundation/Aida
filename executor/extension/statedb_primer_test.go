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

	ext := MakeStateDbPrimer[any](config)
	if _, ok := ext.(NilExtension[any]); !ok {
		t.Errorf("Primer is enabled although not set in configuration")
	}

}

func TestProgressLoggerExtension_PrimingDoesNotTriggerForExistingStateDb(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	config := &utils.Config{}
	config.SkipPriming = false
	config.IsExistingStateDb = true

	log.EXPECT().Warning("Skipping priming due to usage of preexisting StateDb")

	ext := makeStateDbPrimer[any](config, log)

	ext.PreRun(executor.State[any]{}, nil)

}

func TestProgressLoggerExtension_PrimingDoesTriggerForNonExistingStateDb(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	config := &utils.Config{}
	config.SkipPriming = false
	config.StateDbSrc = ""
	config.First = 2

	log.EXPECT().Noticef("Priming to block %v", config.First-1)

	ext := makeStateDbPrimer[any](config, log)

	ext.PreRun(executor.State[any]{}, &executor.Context{})
}

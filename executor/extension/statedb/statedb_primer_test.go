package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestProgressLoggerExtension_NoPrimerIsCreatedIfDisabled(t *testing.T) {
	cfg := &utils.Config{}
	cfg.SkipPriming = true

	ext := MakeStateDbPrimer[any](cfg)
	if _, ok := ext.(extension.NilExtension[any]); !ok {
		t.Errorf("Primer is enabled although not set in configuration")
	}

}

func TestProgressLoggerExtension_PrimingDoesNotTriggerForExistingStateDb(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.SkipPriming = false
	cfg.IsExistingStateDb = true

	log.EXPECT().Warning("Skipping priming due to usage of preexisting StateDb")

	ext := makeStateDbPrimer[any](cfg, log)

	ext.PreRun(executor.State[any]{}, nil)

}

func TestProgressLoggerExtension_PrimingDoesTriggerForNonExistingStateDb(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.SkipPriming = false
	cfg.StateDbSrc = ""
	cfg.First = 2

	log.EXPECT().Noticef("Priming to block %v", cfg.First-1)

	ext := makeStateDbPrimer[any](cfg, log)

	ext.PreRun(executor.State[any]{}, &executor.Context{})
}

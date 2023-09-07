package extension

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"go.uber.org/mock/gomock"
)

func TestTxValidator_NoValidatorIsCreatedIfDisabled(t *testing.T) {
	config := &utils.Config{}
	config.ValidateTxState = false

	ext := MakeTxValidator(config)

	if _, ok := ext.(NilExtension); !ok {
		t.Errorf("Logger is enabled although not set in configuration")
	}
}

func TestTxValidator_ValidatorIsEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	config := &utils.Config{}
	config.ValidateTxState = true

	ext := makeTxValidator(config, log)

	log.EXPECT().Warning(gomock.Any())
	ext.PreRun(executor.State{})
}

func TestTxValidator_ValidatorDoesNotFailWithEmptySubstate(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	config := &utils.Config{}
	config.ValidateTxState = true

	ext := makeTxValidator(config, log)

	log.EXPECT().Warning(gomock.Any())
	ext.PreRun(executor.State{})
	ext.PostTransaction(executor.State{
		Block:       1,
		Transaction: 1,
		Substate:    &substate.Substate{},
		State:       db,
	})
}

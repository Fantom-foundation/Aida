package extension

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
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

	err := ext.PostTransaction(executor.State{
		Block:       1,
		Transaction: 1,
		Substate:    &substate.Substate{},
		State:       db,
	})

	if err != nil {
		t.Errorf("PostTransaction must not return an error!")
	}
}

func TestTxValidator_SingleErrorInPostTransactionDoesNotEndProgramWithNoContinueOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	config := &utils.Config{}
	config.ValidateTxState = true
	config.ContinueOnFailure = true
	config.MaxNumErrors = 2

	ext := MakeTxValidator(config)

	gomock.InOrder(
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(big.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State{})

	err := ext.PreTransaction(executor.State{
		Block:       1,
		Transaction: 1,
		Substate:    getIncorrectTestSubstateAlloc(),
		State:       db,
	})

	if err != nil {
		t.Errorf("PostTransaction must return an error!")
	}
}

func TestTxValidator_SingleErrorInPreTransactionReturnsErrorWithNoContinueOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	config := &utils.Config{}
	config.ValidateTxState = true
	config.ContinueOnFailure = false

	ext := MakeTxValidator(config)

	gomock.InOrder(
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(big.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State{})

	err := ext.PreTransaction(executor.State{
		Block:       1,
		Transaction: 1,
		Substate:    getIncorrectTestSubstateAlloc(),
		State:       db,
	})

	if err == nil {
		t.Errorf("PreTransaction must return an error!")
	}
}

func TestTxValidator_SingleErrorInPostTransactionReturnsErrorWithNoContinueOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	config := &utils.Config{}
	config.ValidateTxState = true
	config.ContinueOnFailure = false

	ext := MakeTxValidator(config)

	gomock.InOrder(
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(big.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State{})

	err := ext.PreTransaction(executor.State{
		Block:       1,
		Transaction: 1,
		Substate:    getIncorrectTestSubstateAlloc(),
		State:       db,
	})

	if err == nil {
		t.Errorf("PostTransaction must return an error!")
	}
}

func TestTxValidator_TwoErrorsDoNotReturnAnErrorWhenContinueOnFailureIsEnabledAndMaxNumErrorsIsHighEnough(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	log := logger.NewMockLogger(ctrl)

	config := &utils.Config{}
	config.ValidateTxState = true
	config.ContinueOnFailure = true
	config.MaxNumErrors = 3

	ext := makeTxValidator(config, log)

	gomock.InOrder(
		// PreRun
		log.EXPECT().Warning(gomock.Any()),
		log.EXPECT().Warningf(gomock.Any(), config.MaxNumErrors),
		// PreTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(big.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
		log.EXPECT().Error(gomock.Any()),
		// PostTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(big.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
		log.EXPECT().Error(gomock.Any()),
	)

	ext.PreRun(executor.State{})

	err := ext.PreTransaction(executor.State{
		Block:       1,
		Transaction: 1,
		Substate:    getIncorrectTestSubstateAlloc(),
		State:       db,
	})

	if err != nil {
		t.Errorf("PreTransaction must not return an error because continue on failure is true!")
	}

	err = ext.PostTransaction(executor.State{
		Block:       1,
		Transaction: 1,
		Substate:    getIncorrectTestSubstateAlloc(),
		State:       db,
	})

	if err != nil {
		t.Errorf("PostTransaction must not return an error because continue on failure is true!")
	}
}

func TestTxValidator_TwoErrorsDoReturnErrorOnEventWhenContinueOnFailureIsEnabledAndMaxNumErrorsIsNotHighEnough(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	log := logger.NewMockLogger(ctrl)

	config := &utils.Config{}
	config.ValidateTxState = true
	config.ContinueOnFailure = true
	config.MaxNumErrors = 2

	ext := makeTxValidator(config, log)

	gomock.InOrder(
		// PreRun
		log.EXPECT().Warning(gomock.Any()),
		log.EXPECT().Warningf(gomock.Any(), config.MaxNumErrors),
		// PreTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(big.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
		log.EXPECT().Error(gomock.Any()),
		// PostTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(big.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
		log.EXPECT().Error(gomock.Any()),
	)

	ext.PreRun(executor.State{})

	err := ext.PreTransaction(executor.State{
		Block:       1,
		Transaction: 1,
		Substate:    getIncorrectTestSubstateAlloc(),
		State:       db,
	})

	if err != nil {
		t.Errorf("PreTransaction must not return an error because continue on failure is true!")
	}

	err = ext.PostTransaction(executor.State{
		Block:       1,
		Transaction: 1,
		Substate:    getIncorrectTestSubstateAlloc(),
		State:       db,
	})

	if err == nil {
		t.Errorf("PostTransaction must return an error because MaxNumErrors is not high enough!")
	}
}

// getIncorrectTestSubstateAlloc returns an error
// Substate with incorrect InputAlloc and OutputAlloc.
// This func is only used in testing.
func getIncorrectTestSubstateAlloc() *substate.Substate {
	sub := &substate.Substate{
		InputAlloc:  make(substate.SubstateAlloc),
		OutputAlloc: make(substate.SubstateAlloc),
	}
	sub.InputAlloc[common.Address{0}] = &substate.SubstateAccount{
		Nonce:   0,
		Balance: new(big.Int),
		Storage: make(map[common.Hash]common.Hash),
		Code:    make([]byte, 0),
	}

	sub.OutputAlloc[common.Address{0}] = &substate.SubstateAccount{
		Nonce:   0,
		Balance: new(big.Int),
		Storage: make(map[common.Hash]common.Hash),
		Code:    make([]byte, 0),
	}

	return sub
}

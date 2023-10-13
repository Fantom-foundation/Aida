package validator

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

const (
	maxNumErrorsTestErr   = "maximum number of errors occurred"
	incorrectInputTestErr = "input error at block 1 tx 1;   Account 0x0000000000000000000000000000000000000000 does not exist\n  " +
		"Failed to validate code for account 0x0000000000000000000000000000000000000000\n    " +
		"have len 1\n    " +
		"want len 0\n"
	incorrectOutputTestErr = "output error at block 1 tx 1;   Account 0x0000000000000000000000000000000000000000 does not exist\n  " +
		"Failed to validate code for account 0x0000000000000000000000000000000000000000\n    " +
		"have len 1\n    " +
		"want len 0\n"
)

func TestTxValidator_NoValidatorIsCreatedIfDisabled(t *testing.T) {
	config := &utils.Config{}
	config.ValidateTxState = false

	ext := MakeTxValidator(config)

	if _, ok := ext.(extension.NilExtension[*substate.Substate]); !ok {
		t.Errorf("Validator is enabled although not set in configuration")
	}
}

func TestTxValidator_ValidatorIsEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	config := &utils.Config{}
	config.ValidateTxState = true

	ext := makeTxValidator(config, log)

	log.EXPECT().Warning(gomock.Any())
	ext.PreRun(executor.State[*substate.Substate]{}, nil)
}

func TestTxValidator_ValidatorDoesNotFailWithEmptySubstate(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)
	context := &executor.Context{State: db}

	config := &utils.Config{}
	config.ValidateTxState = true

	ext := makeTxValidator(config, log)

	log.EXPECT().Warning(gomock.Any())
	ext.PreRun(executor.State[*substate.Substate]{}, context)

	err := ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        &substate.Substate{},
	}, context)

	if err != nil {
		t.Errorf("PostTransaction must not return an error, got %v", err)
	}
}

func TestTxValidator_SingleErrorInPreTransactionDoesNotEndProgramWithContinueOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	context := &executor.Context{State: db}

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

	ext.PreRun(executor.State[*substate.Substate]{}, context)

	err := ext.PreTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, context)

	if err != nil {
		t.Errorf("PreTransaction must not return an error, got %v", err)
	}

	err = ext.PostRun(executor.State[*substate.Substate]{}, context, nil)
	if err == nil {
		t.Fatalf("PostRun must return an error")
	}

	got := err.Error()
	want := incorrectInputTestErr

	if strings.Compare(got, want) != 0 {
		t.Errorf("Unexpected err!\nGot: %v; want: %v", got, want)
	}
}

func TestTxValidator_SingleErrorInPreTransactionReturnsErrorWithNoContinueOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	context := &executor.Context{State: db}

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

	ext.PreRun(executor.State[*substate.Substate]{}, context)

	err := ext.PreTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, context)

	if err == nil {
		t.Errorf("PreTransaction must return an error!")
	}

	err = ext.PostRun(executor.State[*substate.Substate]{}, nil, nil)
	if err == nil {
		t.Errorf("PostRun must return an error!")
	}

	got := err.Error()
	want := incorrectInputTestErr

	if strings.Compare(got, want) != 0 {
		t.Errorf("Unexpected err!\nGot: %v; want: %v", got, want)
	}

}

func TestTxValidator_SingleErrorInPostTransactionReturnsErrorWithNoContinueOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	context := &executor.Context{State: db}

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

	ext.PreRun(executor.State[*substate.Substate]{}, context)

	err := ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, context)

	if err == nil {
		t.Errorf("PreTransaction must return an error!")
	}

	err = ext.PostRun(executor.State[*substate.Substate]{}, nil, nil)
	if err == nil {
		t.Errorf("PostRun must return an error!")
	}

	got := err.Error()
	want := incorrectOutputTestErr

	if strings.Compare(got, want) != 0 {
		t.Errorf("Unexpected err!\nGot: %v; want: %v", got, want)
	}
}

func TestTxValidator_TwoErrorsDoNotReturnAnErrorWhenContinueOnFailureIsEnabledAndMaxNumErrorsIsHighEnough(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	log := logger.NewMockLogger(ctrl)
	context := &executor.Context{State: db}

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
		// PostRun
		log.EXPECT().Warningf(gomock.Any(), 2),
	)

	ext.PreRun(executor.State[*substate.Substate]{}, context)

	err := ext.PreTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, context)

	if err != nil {
		t.Errorf("PreTransaction must not return an error because continue on failure is true!")
	}

	err = ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, context)

	// PostTransaction must not return error because ContinueOnFailure is enabled and error threshold is high enough
	if err != nil {
		t.Errorf("PostTransaction must not return an error because continue on failure is true!")
	}

	// though PostRun must return error because we want to see the errors at the end of the run
	err = ext.PostRun(executor.State[*substate.Substate]{}, context, nil)
	if err == nil {
		t.Errorf("PostRun must return an error!")
	}

	got := err.Error()
	want := errors.Join(errors.New(incorrectInputTestErr), errors.New(incorrectOutputTestErr)).Error()

	if strings.Compare(got, want) != 0 {
		t.Errorf("Unexpected err!\nGot: %v; want: %v", got, want)
	}
}

func TestTxValidator_TwoErrorsDoReturnErrorOnEventWhenContinueOnFailureIsEnabledAndMaxNumErrorsIsNotHighEnough(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	log := logger.NewMockLogger(ctrl)
	context := &executor.Context{State: db}

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
		log.EXPECT().Critical(errors.New("maximum number of errors occurred")),
		// PostRun
		log.EXPECT().Warningf(gomock.Any(), 2),
	)

	ext.PreRun(executor.State[*substate.Substate]{}, context)

	err := ext.PreTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, context)

	if err != nil {
		t.Errorf("PreTransaction must not return an error because continue on failure is true, got %v", err)
	}

	err = ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, context)

	if err == nil {
		t.Errorf("PostTransaction must return an error because MaxNumErrors is not high enough!")
	}

	got := err.Error()
	want := maxNumErrorsTestErr

	if strings.Compare(got, want) != 0 {
		t.Errorf("Unexpected err!\nGot: %v; want: %v", got, want)
	}

	err = ext.PostRun(executor.State[*substate.Substate]{}, context, nil)
	if err == nil {
		t.Errorf("PostRun must return an error because MaxNumErrors is not high enough!")
	}

	got = err.Error()
	want = errors.Join(errors.New(incorrectInputTestErr), errors.New(incorrectOutputTestErr)).Error()

	if strings.Compare(got, want) != 0 {
		t.Errorf("Unexpected err!\nGot: %v; want: %v", got, want)
	}
}

func TestTxValidator_PreTransactionDoesNotFailWithIncorrectOutput(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	context := &executor.Context{State: db}

	config := &utils.Config{}
	config.ValidateTxState = true
	config.ContinueOnFailure = false

	ext := MakeTxValidator(config)

	ext.PreRun(executor.State[*substate.Substate]{}, context)

	err := ext.PreTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data: &substate.Substate{
			OutputAlloc: getIncorrectSubstateAlloc(),
		},
	}, context)

	if err != nil {
		t.Errorf("PreTransaction must not return an error, got %v", err)
	}
}

func TestTxValidator_PostTransactionDoesNotFailWithIncorrectInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	context := &executor.Context{State: db}

	config := &utils.Config{}
	config.ValidateTxState = true
	config.ContinueOnFailure = false

	ext := MakeTxValidator(config)

	ext.PreRun(executor.State[*substate.Substate]{}, context)

	err := ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data: &substate.Substate{
			InputAlloc: getIncorrectSubstateAlloc(),
		},
	}, context)

	if err != nil {
		t.Errorf("PostTransaction must not return an error, got %v", err)
	}
}

// getIncorrectTestSubstateAlloc returns an error
// Substate with incorrect InputAlloc and OutputAlloc.
// This func is only used in testing.
func getIncorrectTestSubstateAlloc() *substate.Substate {
	sub := &substate.Substate{
		InputAlloc:  getIncorrectSubstateAlloc(),
		OutputAlloc: getIncorrectSubstateAlloc(),
	}

	return sub
}

func getIncorrectSubstateAlloc() substate.SubstateAlloc {
	alloc := make(substate.SubstateAlloc)
	alloc[common.Address{0}] = &substate.SubstateAccount{
		Nonce:   0,
		Balance: new(big.Int),
		Storage: make(map[common.Hash]common.Hash),
		Code:    make([]byte, 0),
	}

	return alloc
}

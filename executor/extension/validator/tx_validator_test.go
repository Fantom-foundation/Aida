package validator

import (
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
	incorrectInputTestErr = "input error at block 1 tx 1;   Account 0x0000000000000000000000000000000000000000 does not exist\n  " +
		"Failed to validate code for account 0x0000000000000000000000000000000000000000\n    " +
		"have len 1\n    " +
		"want len 0\n"
	incorrectOutputTestErr = "output error at block 1 tx 1;   Account 0x0000000000000000000000000000000000000000 does not exist\n  " +
		"Failed to validate code for account 0x0000000000000000000000000000000000000000\n    " +
		"have len 1\n    " +
		"want len 0\n"
	incorrectOutputAllocErr = "output error at block 1 tx 1; inconsistent output: alloc"
)

func TestTxValidator_NoValidatorIsCreatedIfDisabled(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ValidateTxState = false

	ext := MakeTxValidator(cfg)

	if _, ok := ext.(extension.NilExtension[*substate.Substate]); !ok {
		t.Errorf("Validator is enabled although not set in configuration")
	}
}

func TestTxValidator_ValidatorIsEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true

	ext := makeTxValidator(cfg, log)

	log.EXPECT().Warning(gomock.Any())
	ext.PreRun(executor.State[*substate.Substate]{}, nil)
}

func TestTxValidator_ValidatorDoesNotFailWithEmptySubstate(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}

	cfg := &utils.Config{}
	cfg.ValidateTxState = true

	ext := makeTxValidator(cfg, log)

	log.EXPECT().Warning(gomock.Any())
	ext.PreRun(executor.State[*substate.Substate]{}, ctx)

	err := ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        &substate.Substate{},
	}, ctx)

	if err != nil {
		t.Errorf("PostTransaction must not return an error, got %v", err)
	}
}

func TestTxValidator_SingleErrorInPreTransactionDoesNotEndProgramWithContinueOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = true
	cfg.MaxNumErrors = 2

	ext := MakeTxValidator(cfg)

	gomock.InOrder(
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(big.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State[*substate.Substate]{}, ctx)

	err := ext.PreTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, ctx)

	if err != nil {
		t.Errorf("PreTransaction must not return an error, got %v", err)
	}
}

func TestTxValidator_SingleErrorInPreTransactionReturnsErrorWithNoContinueOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false

	ext := MakeTxValidator(cfg)

	gomock.InOrder(
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(big.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State[*substate.Substate]{}, ctx)

	err := ext.PreTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, ctx)

	if err == nil {
		t.Errorf("PreTransaction must return an error!")
	}

	got := strings.TrimSpace(err.Error())
	want := strings.TrimSpace(incorrectInputTestErr)

	if strings.Compare(got, want) != 0 {
		t.Errorf("Unexpected err!\nGot: %v; want: %v", got, want)
	}

}

func TestTxValidator_SingleErrorInPostTransactionReturnsErrorWithNoContinueOnFailure_SubsetCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false
	cfg.StateValidationMode = utils.SubsetCheck

	ext := MakeTxValidator(cfg)

	gomock.InOrder(
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().CreateAccount(common.Address{0}),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(big.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
		db.EXPECT().SetCode(common.Address{0}, []byte{}),
	)

	ext.PreRun(executor.State[*substate.Substate]{}, ctx)

	err := ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, ctx)

	if err == nil {
		t.Errorf("PostTransaction must return an error!")
	}

	got := strings.TrimSpace(err.Error())
	want := strings.TrimSpace(incorrectOutputTestErr)

	if strings.Compare(got, want) != 0 {
		t.Errorf("Unexpected err!\nGot: %v; \nWant: %v", got, want)
	}
}

func TestTxValidator_SingleErrorInPostTransactionReturnsErrorWithNoContinueOnFailure_EqualityCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false
	cfg.StateValidationMode = utils.EqualityCheck

	ext := MakeTxValidator(cfg)

	db.EXPECT().GetSubstatePostAlloc().Return(substate.SubstateAlloc{})

	ext.PreRun(executor.State[*substate.Substate]{}, ctx)

	err := ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, ctx)

	if err == nil {
		t.Fatal("PostTransaction must return an error!")
	}

	got := strings.TrimSpace(err.Error())
	want := strings.TrimSpace(incorrectOutputAllocErr)

	if strings.Compare(got, want) != 0 {
		t.Errorf("Unexpected err!\nGot: %v; \nWant: %v", got, want)
	}
}

func TestTxValidator_TwoErrorsDoNotReturnAnErrorWhenContinueOnFailureIsEnabledAndMaxNumErrorsIsHighEnough(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	log := logger.NewMockLogger(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = true
	cfg.MaxNumErrors = 3

	ext := makeTxValidator(cfg, log)

	gomock.InOrder(
		// PreRun
		log.EXPECT().Warning(gomock.Any()),
		log.EXPECT().Warningf(gomock.Any(), cfg.MaxNumErrors),
		// PreTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(big.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
		// PostTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(big.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State[*substate.Substate]{}, ctx)

	err := ext.PreTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, ctx)

	if err != nil {
		t.Errorf("PreTransaction must not return an error because continue on failure is true!")
	}

	err = ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, ctx)

	// PostTransaction must not return error because ContinueOnFailure is enabled and error threshold is high enough
	if err != nil {
		t.Errorf("PostTransaction must not return an error because continue on failure is true!")
	}
}

func TestTxValidator_TwoErrorsDoReturnErrorOnEventWhenContinueOnFailureIsEnabledAndMaxNumErrorsIsNotHighEnough(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	log := logger.NewMockLogger(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = true
	cfg.MaxNumErrors = 2

	ext := makeTxValidator(cfg, log)

	gomock.InOrder(
		// PreRun
		log.EXPECT().Warning(gomock.Any()),
		log.EXPECT().Warningf(gomock.Any(), cfg.MaxNumErrors),
		// PreTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(big.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
		// PostTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().CreateAccount(common.Address{0}),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(big.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
		db.EXPECT().SetCode(common.Address{0}, []byte{}),
	)

	ext.PreRun(executor.State[*substate.Substate]{}, ctx)

	err := ext.PreTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, ctx)

	if err != nil {
		t.Errorf("PreTransaction must not return an error because continue on failure is true, got %v", err)
	}

	err = ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestSubstateAlloc(),
	}, ctx)

	if err == nil {
		t.Errorf("PostTransaction must return an error because MaxNumErrors is not high enough!")
	}
}

func TestTxValidator_PreTransactionDoesNotFailWithIncorrectOutput(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false

	ext := MakeTxValidator(cfg)

	ext.PreRun(executor.State[*substate.Substate]{}, ctx)

	err := ext.PreTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data: &substate.Substate{
			OutputAlloc: getIncorrectSubstateAlloc(),
		},
	}, ctx)

	if err != nil {
		t.Errorf("PreTransaction must not return an error, got %v", err)
	}
}

func TestTxValidator_PostTransactionDoesNotFailWithIncorrectInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false

	ext := MakeTxValidator(cfg)

	ext.PreRun(executor.State[*substate.Substate]{}, ctx)

	err := ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data: &substate.Substate{
			InputAlloc: getIncorrectSubstateAlloc(),
		},
	}, ctx)

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

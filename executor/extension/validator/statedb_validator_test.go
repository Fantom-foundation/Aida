package validator

import (
	"bytes"
	"fmt"
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
	incorrectInputTestErr  = "block 1 tx 1\n input alloc is not contained in the state-db\n   Account 0x0000000000000000000000000000000000000000 does not exist\n  Failed to validate code for account 0x0000000000000000000000000000000000000000\n    have len 1\n    want len 0"
	incorrectOutputTestErr = "output error at block 1 tx 1;   Account 0x0000000000000000000000000000000000000000 does not exist\n  " +
		"Failed to validate code for account 0x0000000000000000000000000000000000000000\n    " +
		"have len 1\n    " +
		"want len 0\n"
	incorrectOutputAllocErr = "output error at block 1 tx 1; inconsistent output: alloc"
)

func TestLiveTxValidator_NoValidatorIsCreatedIfDisabled(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ValidateTxState = false

	ext := MakeLiveDbValidator(cfg)

	if _, ok := ext.(extension.NilExtension[*substate.Substate]); !ok {
		t.Errorf("Validator is enabled although not set in configuration")
	}
}

func TestLiveTxValidator_ValidatorIsEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true

	ext := makeLiveDbValidator(cfg, log)

	log.EXPECT().Warning(gomock.Any())
	ext.PreRun(executor.State[*substate.Substate]{}, nil)
}

func TestLiveTxValidator_ValidatorDoesNotFailWithEmptySubstate(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}

	cfg := &utils.Config{}
	cfg.ValidateTxState = true

	ext := makeLiveDbValidator(cfg, log)

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

func TestLiveTxValidator_SingleErrorInPreTransactionDoesNotEndProgramWithContinueOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = true
	cfg.MaxNumErrors = 2

	ext := MakeLiveDbValidator(cfg)

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

func TestLiveTxValidator_SingleErrorInPreTransactionReturnsErrorWithNoContinueOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false

	ext := MakeLiveDbValidator(cfg)

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

func TestLiveTxValidator_SingleErrorInPostTransactionReturnsErrorWithNoContinueOnFailure_SubsetCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false
	cfg.StateValidationMode = utils.SubsetCheck

	ext := MakeLiveDbValidator(cfg)

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

func TestLiveTxValidator_SingleErrorInPostTransactionReturnsErrorWithNoContinueOnFailure_EqualityCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false
	cfg.StateValidationMode = utils.EqualityCheck

	ext := MakeLiveDbValidator(cfg)

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

func TestLiveTxValidator_TwoErrorsDoNotReturnAnErrorWhenContinueOnFailureIsEnabledAndMaxNumErrorsIsHighEnough(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	log := logger.NewMockLogger(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = true
	cfg.MaxNumErrors = 3

	ext := makeLiveDbValidator(cfg, log)

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

func TestLiveTxValidator_TwoErrorsDoReturnErrorOnEventWhenContinueOnFailureIsEnabledAndMaxNumErrorsIsNotHighEnough(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	log := logger.NewMockLogger(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = true
	cfg.MaxNumErrors = 2

	ext := makeLiveDbValidator(cfg, log)

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

func TestLiveTxValidator_PreTransactionDoesNotFailWithIncorrectOutput(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false

	ext := MakeLiveDbValidator(cfg)

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

func TestLiveTxValidator_PostTransactionDoesNotFailWithIncorrectInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false

	ext := MakeLiveDbValidator(cfg)

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

func TestArchiveTxValidator_NoValidatorIsCreatedIfDisabled(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ValidateTxState = false

	ext := MakeArchiveDbValidator(cfg)

	if _, ok := ext.(extension.NilExtension[*substate.Substate]); !ok {
		t.Errorf("Validator is enabled although not set in configuration")
	}
}

func TestArchiveTxValidator_ValidatorIsEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true

	ext := makeArchiveDbValidator(cfg, log)

	log.EXPECT().Warning(gomock.Any())
	ext.PreRun(executor.State[*substate.Substate]{}, nil)
}

func TestArchiveTxValidator_ValidatorDoesNotFailWithEmptySubstate(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockNonCommittableStateDB(ctrl)
	ctx := &executor.Context{Archive: db}

	cfg := &utils.Config{}
	cfg.ValidateTxState = true

	ext := makeArchiveDbValidator(cfg, log)

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

func TestArchiveTxValidator_SingleErrorInPreTransactionDoesNotEndProgramWithContinueOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockNonCommittableStateDB(ctrl)
	ctx := &executor.Context{Archive: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = true
	cfg.MaxNumErrors = 2

	ext := MakeArchiveDbValidator(cfg)

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

func TestArchiveTxValidator_SingleErrorInPreTransactionReturnsErrorWithNoContinueOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockNonCommittableStateDB(ctrl)
	ctx := &executor.Context{Archive: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false

	ext := MakeArchiveDbValidator(cfg)

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

func TestArchiveTxValidator_SingleErrorInPostTransactionReturnsErrorWithNoContinueOnFailure_SubsetCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockNonCommittableStateDB(ctrl)
	ctx := &executor.Context{Archive: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false
	cfg.StateValidationMode = utils.SubsetCheck

	ext := MakeArchiveDbValidator(cfg)

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

func TestArchiveTxValidator_SingleErrorInPostTransactionReturnsErrorWithNoContinueOnFailure_EqualityCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockNonCommittableStateDB(ctrl)
	ctx := &executor.Context{Archive: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false
	cfg.StateValidationMode = utils.EqualityCheck

	ext := MakeArchiveDbValidator(cfg)

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

func TestArchiveTxValidator_TwoErrorsDoNotReturnAnErrorWhenContinueOnFailureIsEnabledAndMaxNumErrorsIsHighEnough(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockNonCommittableStateDB(ctrl)
	log := logger.NewMockLogger(ctrl)
	ctx := &executor.Context{Archive: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = true
	cfg.MaxNumErrors = 3

	ext := makeArchiveDbValidator(cfg, log)

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

func TestArchiveTxValidator_TwoErrorsDoReturnErrorOnEventWhenContinueOnFailureIsEnabledAndMaxNumErrorsIsNotHighEnough(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockNonCommittableStateDB(ctrl)
	log := logger.NewMockLogger(ctrl)
	ctx := &executor.Context{Archive: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = true
	cfg.MaxNumErrors = 2

	ext := makeArchiveDbValidator(cfg, log)

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

func TestArchiveTxValidator_PreTransactionDoesNotFailWithIncorrectOutput(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockNonCommittableStateDB(ctrl)
	ctx := &executor.Context{Archive: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false

	ext := MakeArchiveDbValidator(cfg)

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

func TestArchiveTxValidator_PostTransactionDoesNotFailWithIncorrectInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockNonCommittableStateDB(ctrl)
	ctx := &executor.Context{Archive: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false

	ext := MakeLiveDbValidator(cfg)

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

// TestStateDb_ValidateStateDB tests validation of state DB by comparing it to valid world state
func TestValidateStateDb_ValidationDoesNotFail(t *testing.T) {
	for _, tc := range utils.GetStateDbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.Variant, tc.ShadowImpl, tc.ArchiveVariant), func(t *testing.T) {
			cfg := utils.MakeTestConfig(tc)

			// Initialization of state DB
			sDB, _, err := utils.PrepareStateDB(cfg)
			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			// Closing of state DB
			defer func(sDB state.StateDB) {
				err = sDB.Close()
				if err != nil {
					t.Fatalf("failed to close state DB: %v", err)
				}
			}(sDB)

			// Generating randomized world state
			ws, _ := utils.MakeWorldState(t)

			log := logger.NewLogger("INFO", "TestStateDb")

			// Create new prime context
			pc := utils.NewPrimeContext(cfg, sDB, log)
			// Priming state DB with given world state
			pc.PrimeStateDB(ws, sDB)

			// Call for state DB validation and subsequent check for error
			err = validateStateDb(ws, sDB, false)
			if err != nil {
				t.Fatalf("failed to validate state DB: %v", err)
			}
		})
	}
}

// TestStateDb_ValidateStateDBWithUpdate test state DB validation comparing it to valid world state
// given state DB should be updated if world state contains different data
func TestValidateStateDb_ValidationDoesNotFailWithPriming(t *testing.T) {
	for _, tc := range utils.GetStateDbTestCases() {
		t.Run(fmt.Sprintf("DB variant: %s; shadowImpl: %s; archive variant: %s", tc.Variant, tc.ShadowImpl, tc.ArchiveVariant), func(t *testing.T) {
			cfg := utils.MakeTestConfig(tc)

			// Initialization of state DB
			sDB, _, err := utils.PrepareStateDB(cfg)
			if err != nil {
				t.Fatalf("failed to create state DB: %v", err)
			}

			// Closing of state DB
			defer func(sDB state.StateDB) {
				err = sDB.Close()
				if err != nil {
					t.Fatalf("failed to close state DB: %v", err)
				}
			}(sDB)

			// Generating randomized world state
			ws, _ := utils.MakeWorldState(t)

			log := logger.NewLogger("INFO", "TestStateDb")

			// Create new prime context
			pc := utils.NewPrimeContext(cfg, sDB, log)
			// Priming state DB with given world state
			pc.PrimeStateDB(ws, sDB)

			// create new random address
			addr := common.BytesToAddress(utils.MakeRandomByteSlice(t, 40))

			// create new account
			ws[addr] = &substate.SubstateAccount{
				Nonce:   uint64(utils.GetRandom(1, 1000*5000)),
				Balance: big.NewInt(int64(utils.GetRandom(1, 1000*5000))),
				Storage: utils.MakeAccountStorage(t),
				Code:    utils.MakeRandomByteSlice(t, 2048),
			}

			// Call for state DB validation with update enabled and subsequent checks if the update was made correctly
			err = validateStateDb(ws, sDB, true)
			if err == nil {
				t.Fatalf("failed to throw errors while validating state DB: %v", err)
			}

			if sDB.GetBalance(addr).Cmp(ws[addr].Balance) != 0 {
				t.Fatalf("failed to prime account balance; Is: %v; Should be: %v", sDB.GetBalance(addr), ws[addr].Balance)
			}

			if sDB.GetNonce(addr) != ws[addr].Nonce {
				t.Fatalf("failed to prime account nonce; Is: %v; Should be: %v", sDB.GetNonce(addr), ws[addr].Nonce)
			}

			if bytes.Compare(sDB.GetCode(addr), ws[addr].Code) != 0 {
				t.Fatalf("failed to prime account code; Is: %v; Should be: %v", sDB.GetCode(addr), ws[addr].Code)
			}

			for sKey, sValue := range ws[addr].Storage {
				if sDB.GetState(addr, sKey) != sValue {
					t.Fatalf("failed to prime account storage; Is: %v; Should be: %v", sDB.GetState(addr, sKey), sValue)
				}
			}
		})
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

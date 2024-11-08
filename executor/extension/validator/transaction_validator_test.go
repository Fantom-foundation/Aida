// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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
	"github.com/Fantom-foundation/Aida/txcontext"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/substate"
	substatetypes "github.com/Fantom-foundation/Substate/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
)

const (
	liveDbIncorrectInputTestErr  = "live-db-validator err:\nblock 1 tx 1\n world-state input is not contained in the state-db\n   Account 0x0000000000000000000000000000000000000000 does not exist\n  Failed to validate code for account 0x0000000000000000000000000000000000000000\n    have len 1\n    want len 0"
	liveDbIncorrectOutputTestErr = "live-db-validator err:\nworld-state output error at block 1 tx 1;   Account 0x0000000000000000000000000000000000000000 does not exist\n  " +
		"Failed to validate code for account 0x0000000000000000000000000000000000000000\n    " +
		"have len 1\n    " +
		"want len 0\n"
	liveDbIncorrectOutputAllocErr = "live-db-validator err:\nworld-state output error at block 1 tx 1; inconsistent output: alloc"

	archiveDbIncorrectInputTestErr  = "archive-db-validator err:\nblock 1 tx 1\n world-state input is not contained in the state-db\n   Account 0x0000000000000000000000000000000000000000 does not exist\n  Failed to validate code for account 0x0000000000000000000000000000000000000000\n    have len 1\n    want len 0"
	archiveDbIncorrectOutputTestErr = "archive-db-validator err:\nworld-state output error at block 1 tx 1;   Account 0x0000000000000000000000000000000000000000 does not exist\n  " +
		"Failed to validate code for account 0x0000000000000000000000000000000000000000\n    " +
		"have len 1\n    " +
		"want len 0\n"
	archiveDbIncorrectOutputAllocErr = "archive-db-validator err:\nworld-state output error at block 1 tx 1; inconsistent output: alloc"
)

func TestLiveTxValidator_NoValidatorIsCreatedIfDisabled(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ValidateTxState = false

	ext := MakeLiveDbValidator(cfg, ValidateTxTarget{WorldState: true, Receipt: false})

	if _, ok := ext.(extension.NilExtension[txcontext.TxContext]); !ok {
		t.Errorf("Validator is enabled although not set in configuration")
	}
}

func TestLiveTxValidator_ValidatorIsEnabledWhenOnlyWorldStateIsTested(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true

	ext := makeLiveDbValidator(cfg, log, ValidateTxTarget{WorldState: true, Receipt: false})

	log.EXPECT().Warning(gomock.Any())
	ext.PreRun(executor.State[txcontext.TxContext]{}, nil)
}

func TestLiveTxValidator_ValidatorIsEnabledWhenOnlyReceiptIsTested(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true

	ext := makeLiveDbValidator(cfg, log, ValidateTxTarget{WorldState: false, Receipt: false})

	log.EXPECT().Warning(gomock.Any())
	ext.PreRun(executor.State[txcontext.TxContext]{}, nil)
}

func TestLiveTxValidator_ValidatorIsEnabledWhenBothAreTested(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true

	ext := makeLiveDbValidator(cfg, log, ValidateTxTarget{WorldState: true, Receipt: false})

	log.EXPECT().Warning(gomock.Any())
	ext.PreRun(executor.State[txcontext.TxContext]{}, nil)
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

	ext := MakeLiveDbValidator(cfg, ValidateTxTarget{WorldState: true, Receipt: false})

	gomock.InOrder(
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(uint256.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PreTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
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

	ext := MakeLiveDbValidator(cfg, ValidateTxTarget{WorldState: true, Receipt: false})

	gomock.InOrder(
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(uint256.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PreTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
	}, ctx)

	if err == nil {
		t.Errorf("PreTransaction must return an error!")
	}

	got := strings.TrimSpace(err.Error())
	want := strings.TrimSpace(liveDbIncorrectInputTestErr)

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

	ext := MakeLiveDbValidator(cfg, ValidateTxTarget{WorldState: true, Receipt: false})

	gomock.InOrder(
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(uint256.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PostTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
	}, ctx)

	if err == nil {
		t.Errorf("PostTransaction must return an error!")
	}

	got := strings.TrimSpace(err.Error())
	want := strings.TrimSpace(liveDbIncorrectOutputTestErr)

	if strings.Compare(got, want) != 0 {
		t.Errorf("Unexpected err!\nGot: %v; \nWant: %v", got, want)
	}
}

func TestLiveTxValidator_SingleErrorInPostTransactionReturnsErrorWithNoContinueOnFailure_EqualityCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	log := logger.NewMockLogger(ctrl)
	ctx := &executor.Context{State: db}
	ctx.ErrorInput = make(chan error, 10)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true
	cfg.ContinueOnFailure = false
	cfg.StateValidationMode = utils.EqualityCheck

	ext := makeLiveDbValidator(cfg, log, ValidateTxTarget{WorldState: true, Receipt: false})

	gomock.InOrder(
		log.EXPECT().Warning(gomock.Any()),
		db.EXPECT().GetSubstatePostAlloc().Return(substatecontext.NewWorldState(substate.WorldState{})),
		log.EXPECT().Errorf("Different %s:\nwant: %v\nhave: %v\n", "substate alloc size", 1, 0),
		log.EXPECT().Errorf("\tmissing address=%v\n", common.Address{0}),
	)

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PostTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
	}, ctx)

	if err == nil {
		t.Fatal("PostTransaction must return an error!")
	}

	got := strings.TrimSpace(err.Error())
	want := strings.TrimSpace(liveDbIncorrectOutputAllocErr)

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

	ext := makeLiveDbValidator(cfg, log, ValidateTxTarget{WorldState: true, Receipt: false})

	gomock.InOrder(
		// PreRun
		log.EXPECT().Warning(gomock.Any()),
		log.EXPECT().Warningf(gomock.Any(), cfg.MaxNumErrors),
		// PreTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(uint256.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
		// PostTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(uint256.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PreTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
	}, ctx)

	if err != nil {
		t.Errorf("PreTransaction must not return an error because continue on failure is true!")
	}

	err = ext.PostTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
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

	ext := makeLiveDbValidator(cfg, log, ValidateTxTarget{WorldState: true, Receipt: false})

	gomock.InOrder(
		// PreRun
		log.EXPECT().Warning(gomock.Any()),
		log.EXPECT().Warningf(gomock.Any(), cfg.MaxNumErrors),
		// PreTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(uint256.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
		// PostTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(uint256.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PreTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
	}, ctx)

	if err != nil {
		t.Errorf("PreTransaction must not return an error because continue on failure is true, got %v", err)
	}

	err = ext.PostTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
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

	ext := MakeLiveDbValidator(cfg, ValidateTxTarget{WorldState: true, Receipt: false})

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	alloc := &substate.Substate{
		OutputSubstate: getIncorrectWorldState(),
	}

	err := ext.PreTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        substatecontext.NewTxContext(alloc),
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

	ext := MakeLiveDbValidator(cfg, ValidateTxTarget{WorldState: true, Receipt: false})

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	alloc := &substate.Substate{
		InputSubstate: getIncorrectWorldState(),
	}

	err := ext.PostTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        substatecontext.NewTxContext(alloc),
	}, ctx)

	if err != nil {
		t.Errorf("PostTransaction must not return an error, got %v", err)
	}
}

func TestArchiveTxValidator_NoValidatorIsCreatedIfDisabled(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ValidateTxState = false

	ext := MakeArchiveDbValidator(cfg, ValidateTxTarget{WorldState: true, Receipt: false})

	if _, ok := ext.(extension.NilExtension[txcontext.TxContext]); !ok {
		t.Errorf("Validator is enabled although not set in configuration")
	}
}

func TestArchiveTxValidator_ValidatorIsEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.ValidateTxState = true

	ext := makeArchiveDbValidator(cfg, log, ValidateTxTarget{WorldState: true, Receipt: false})

	log.EXPECT().Warning(gomock.Any())
	ext.PreRun(executor.State[txcontext.TxContext]{}, nil)
}

func TestArchiveTxValidator_ValidatorDoesNotFailWithEmptySubstate(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockNonCommittableStateDB(ctrl)
	ctx := &executor.Context{Archive: db}

	cfg := &utils.Config{}
	cfg.ValidateTxState = true

	ext := makeArchiveDbValidator(cfg, log, ValidateTxTarget{WorldState: true, Receipt: false})

	log.EXPECT().Warning(gomock.Any())
	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PostTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        substatecontext.NewTxContext(&substate.Substate{}),
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

	ext := MakeArchiveDbValidator(cfg, ValidateTxTarget{WorldState: true, Receipt: false})

	gomock.InOrder(
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(uint256.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PreTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
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

	ext := MakeArchiveDbValidator(cfg, ValidateTxTarget{WorldState: true, Receipt: false})

	gomock.InOrder(
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(uint256.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PreTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
	}, ctx)

	if err == nil {
		t.Errorf("PreTransaction must return an error!")
	}

	got := strings.TrimSpace(err.Error())
	want := strings.TrimSpace(archiveDbIncorrectInputTestErr)

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

	ext := MakeArchiveDbValidator(cfg, ValidateTxTarget{WorldState: true, Receipt: false})

	gomock.InOrder(
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(uint256.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PostTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
	}, ctx)

	if err == nil {
		t.Errorf("PostTransaction must return an error!")
	}

	got := strings.TrimSpace(err.Error())
	want := strings.TrimSpace(archiveDbIncorrectOutputTestErr)

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

	ext := MakeArchiveDbValidator(cfg, ValidateTxTarget{WorldState: true, Receipt: false})

	db.EXPECT().GetSubstatePostAlloc().Return(substatecontext.NewWorldState(substate.WorldState{}))

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PostTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
	}, ctx)

	if err == nil {
		t.Fatal("PostTransaction must return an error!")
	}

	got := strings.TrimSpace(err.Error())
	want := strings.TrimSpace(archiveDbIncorrectOutputAllocErr)

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

	ext := makeArchiveDbValidator(cfg, log, ValidateTxTarget{WorldState: true, Receipt: false})

	gomock.InOrder(
		// PreRun
		log.EXPECT().Warning(gomock.Any()),
		log.EXPECT().Warningf(gomock.Any(), cfg.MaxNumErrors),
		// PreTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(uint256.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
		// PostTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(uint256.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PreTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
	}, ctx)

	if err != nil {
		t.Errorf("PreTransaction must not return an error because continue on failure is true!")
	}

	err = ext.PostTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
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

	ext := makeArchiveDbValidator(cfg, log, ValidateTxTarget{WorldState: true, Receipt: false})

	gomock.InOrder(
		// PreRun
		log.EXPECT().Warning(gomock.Any()),
		log.EXPECT().Warningf(gomock.Any(), cfg.MaxNumErrors),
		// PreTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(uint256.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
		// PostTransaction
		db.EXPECT().Exist(common.Address{0}).Return(false),
		db.EXPECT().GetBalance(common.Address{0}).Return(new(uint256.Int)),
		db.EXPECT().GetNonce(common.Address{0}).Return(uint64(0)),
		db.EXPECT().GetCode(common.Address{0}).Return([]byte{0}),
	)

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PreTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
	}, ctx)

	if err != nil {
		t.Errorf("PreTransaction must not return an error because continue on failure is true, got %v", err)
	}

	err = ext.PostTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data:        getIncorrectTestWorldState(),
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

	ext := MakeArchiveDbValidator(cfg, ValidateTxTarget{WorldState: true, Receipt: false})

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PreTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data: substatecontext.NewTxContext(&substate.Substate{
			OutputSubstate: getIncorrectWorldState(),
		}),
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

	ext := MakeLiveDbValidator(cfg, ValidateTxTarget{WorldState: true, Receipt: false})

	ext.PreRun(executor.State[txcontext.TxContext]{}, ctx)

	err := ext.PostTransaction(executor.State[txcontext.TxContext]{
		Block:       1,
		Transaction: 1,
		Data: substatecontext.NewTxContext(&substate.Substate{
			InputSubstate: getIncorrectWorldState(),
		}),
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
				if err = state.CloseCarmenDbTestContext(sDB); err != nil {
					t.Fatal(err)
				}
			}(sDB)

			// Generating randomized world state
			ws, _ := utils.MakeWorldState(t)

			log := logger.NewLogger("INFO", "TestStateDb")

			// Create new prime context
			pc := utils.NewPrimeContext(cfg, sDB, 0, log)
			// Priming state DB with given world state
			if err = pc.PrimeStateDB(ws, sDB); err != nil {
				t.Fatal(err)
			}

			err = sDB.BeginBlock(uint64(2))
			if err != nil {
				t.Fatalf("cannot begin block; %v", err)
			}

			err = sDB.BeginTransaction(uint32(0))
			if err != nil {
				t.Fatalf("cannot begin transaction; %v", err)
			}

			// Call for state DB validation and subsequent check for error
			err = doSubsetValidation(ws, sDB, false)
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
				if err = state.CloseCarmenDbTestContext(sDB); err != nil {
					t.Fatal(err)
				}
			}(sDB)

			// Generating randomized world state
			ws, _ := utils.MakeWorldState(t)

			log := logger.NewLogger("INFO", "TestStateDb")

			// Create new prime context
			pc := utils.NewPrimeContext(cfg, sDB, 0, log)
			// Priming state DB with given world state
			pc.PrimeStateDB(ws, sDB)

			// create new random address
			addr := common.BytesToAddress(utils.MakeRandomByteSlice(t, 40))

			// create new account
			subAcc := txcontext.NewAccount(
				utils.MakeRandomByteSlice(t, 2048),
				utils.MakeAccountStorage(t),
				big.NewInt(int64(utils.GetRandom(1, 1000*5000))),
				uint64(utils.GetRandom(1, 1000*5000)),
			)

			ws[addr] = subAcc

			err = sDB.BeginBlock(uint64(2))
			if err != nil {
				t.Fatalf("cannot begin block; %v", err)
			}
			err = sDB.BeginTransaction(uint32(0))
			if err != nil {
				t.Fatalf("cannot begin transaction; %v", err)
			}

			// Call for state DB validation with update enabled and subsequent checks if the update was made correctly
			err = doSubsetValidation(ws, sDB, true)
			if err == nil {
				t.Fatalf("failed to throw errors while validating state DB: %v", err)
			}

			acc := ws[addr]
			if sDB.GetBalance(addr).Cmp(acc.GetBalance()) != 0 {
				t.Fatalf("failed to prime account balance; Is: %v; Should be: %v", sDB.GetBalance(addr), acc.GetBalance())
			}

			if sDB.GetNonce(addr) != acc.GetNonce() {
				t.Fatalf("failed to prime account nonce; Is: %v; Should be: %v", sDB.GetNonce(addr), acc.GetNonce())
			}

			if bytes.Compare(sDB.GetCode(addr), acc.GetCode()) != 0 {
				t.Fatalf("failed to prime account code; Is: %v; Should be: %v", sDB.GetCode(addr), acc.GetCode())
			}

			acc.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
				if sDB.GetState(addr, keyHash) != valueHash {
					t.Fatalf("failed to prime account storage; Is: %v; Should be: %v", sDB.GetState(addr, common.Hash(keyHash)), valueHash)
				}
			})

		})
	}
}

// TestValidateVMResult tests validatation of data result.
func TestValidateStateDb_ValidateReceipt(t *testing.T) {
	sub := &substate.Substate{Result: getDummyResult()}
	ctx := new(executor.Context)
	ctx.ExecutionResult = substatecontext.NewReceipt(getDummyResult())

	cfg := &utils.Config{}
	cfg.ValidateTxState = true

	ext := makeLiveDbValidator(cfg, logger.NewMockLogger(gomock.NewController(t)), ValidateTxTarget{WorldState: false, Receipt: true})

	// test positive
	err := ext.PostTransaction(executor.State[txcontext.TxContext]{Data: substatecontext.NewTxContext(sub)}, ctx)
	if err != nil {
		t.Fatalf("Failed to validate VM output. %v", err)
	}

	// test negative
	// mismatched contract
	sub.Result.ContractAddress = substatetypes.HexToAddress("0x0000000000085a12481aEdb59eb3200332aCA542")
	err = ext.PostTransaction(executor.State[txcontext.TxContext]{Data: substatecontext.NewTxContext(sub)}, ctx)
	if err == nil {
		t.Fatalf("Failed to validate VM output. Expect contract address mismatch error.")
	}
	// mismatched gas used
	sub = &substate.Substate{Result: getDummyResult()}
	sub.Result.GasUsed = 0
	err = ext.PostTransaction(executor.State[txcontext.TxContext]{Data: substatecontext.NewTxContext(sub)}, ctx)
	if err == nil {
		t.Fatalf("Failed to validate VM output. Expect gas used mismatch error.")
	}

	// mismatched gas used
	sub = &substate.Substate{Result: getDummyResult()}
	sub.Result.Status = types.ReceiptStatusFailed
	err = ext.PostTransaction(executor.State[txcontext.TxContext]{Data: substatecontext.NewTxContext(sub)}, ctx)
	if err == nil {
		t.Fatalf("Failed to validate VM output. Expect staatus mismatch error.")
	}
}

func TestValidateVMResult_ErrorIsInCorrectFormat(t *testing.T) {
	expectedResult := getDummyResult()
	vmResult := getDummyResult()

	cfg := &utils.Config{}
	cfg.ValidateTxState = true

	ext := makeStateDbValidator(cfg, logger.NewMockLogger(gomock.NewController(t)), ValidateTxTarget{WorldState: true, Receipt: false})

	// change result so validation fails
	expectedResult.GasUsed = 15000

	vmRes := substatecontext.NewReceipt(vmResult)
	expRes := substatecontext.NewReceipt(expectedResult)

	err := ext.validateReceipt(vmRes, expRes)
	if err == nil {
		t.Fatal("validation must fail")
	}

	want := fmt.Sprintf(
		"\ngot:\n"+
			"\tstatus: %v\n"+
			"\tbloom: %v\n"+
			"\tlogs: %v\n"+
			"\tcontract address: %v\n"+
			"\tgas used: %v\n"+
			"\nwant:\n"+
			"\tstatus: %v\n"+
			"\tbloom: %v\n"+
			"\tlogs: %v\n"+
			"\tcontract address: %v\n"+
			"\tgas used: %v\n",
		vmRes.GetStatus(),
		vmRes.GetBloom().Big().Uint64(),
		vmRes.GetLogs(),
		vmRes.GetContractAddress(),
		vmRes.GetGasUsed(),
		expRes.GetStatus(),
		expRes.GetBloom().Big().Uint64(),
		expRes.GetLogs(),
		expRes.GetContractAddress(),
		expRes.GetGasUsed(),
	)
	got := err.Error()

	if strings.Compare(got, want) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v\n", got, want)
	}
}

// getIncorrectTestWorldState returns an error
// Substate with incorrect InputSubstate and OutputAlloc.
// This func is only used in testing.
func getIncorrectTestWorldState() txcontext.TxContext {
	sub := &substate.Substate{
		InputSubstate:  getIncorrectWorldState(),
		OutputSubstate: getIncorrectWorldState(),
	}

	return substatecontext.NewTxContext(sub)
}

func getIncorrectWorldState() substate.WorldState {
	alloc := make(substate.WorldState)
	alloc[substatetypes.Address{0}] = &substate.Account{
		Nonce:   0,
		Balance: new(big.Int),
		Storage: make(map[substatetypes.Hash]substatetypes.Hash),
		Code:    make([]byte, 0),
	}

	return alloc
}

func getDummyResult() *substate.Result {
	r := &substate.Result{
		Logs:            []*substatetypes.Log{},
		ContractAddress: substatetypes.HexToAddress("0x0000000000085a12481aEdb59eb3200332aCA541"),
		GasUsed:         1000000,
		Status:          types.ReceiptStatusSuccessful,
	}
	return r
}

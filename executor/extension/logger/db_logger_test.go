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

package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
)

var testAddr = common.Address{0}

func TestDbLoggerExtension_CorrectClose(t *testing.T) {
	cfg := &utils.Config{}
	ext := MakeDbLogger[any](cfg)

	// start the report thread
	ext.PreRun(executor.State[any]{}, nil)

	// make sure PostRun is not blocking.
	done := make(chan bool)
	go func() {
		ext.PostRun(executor.State[any]{}, nil, nil)
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(time.Second):
		t.Fatalf("PostRun blocked unexpectedly")
	}
}

func TestDbLoggerExtension_NoLoggerIsCreatedIfNotEnabled(t *testing.T) {
	cfg := &utils.Config{}
	ext := MakeDbLogger[any](cfg)
	if _, ok := ext.(extension.NilExtension[any]); !ok {
		t.Errorf("Logger is enabled although not set in configuration")
	}

}

func TestDbLoggerExtension_LoggingHappens(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	fileName := t.TempDir() + "test-log"
	cfg := &utils.Config{}
	cfg.DbLogging = fileName

	ext := makeDbLogger[any](cfg, log)

	ctx := &executor.Context{State: db}

	err := ext.PreRun(executor.State[any]{}, ctx)
	if err != nil {
		t.Fatalf("pre-run returned err; %v", err)
	}

	err = ext.PreTransaction(executor.State[any]{}, ctx)
	if err != nil {
		t.Fatalf("pre-transaction returned err; %v", err)
	}

	balance := new(uint256.Int).SetUint64(10)

	beginBlock := fmt.Sprintf("BeginBlock, %v", 1)
	beginTransaction := fmt.Sprintf("BeginTransaction, %v", 0)
	getBalance := fmt.Sprintf("GetBalance, %v, %v", testAddr, balance)
	endTransaction := fmt.Sprintf("EndTransaction")
	endBlock := fmt.Sprintf("EndBlock")

	gomock.InOrder(
		log.EXPECT().Debug(beginBlock),
		db.EXPECT().BeginBlock(uint64(1)),
		log.EXPECT().Debug(beginTransaction),
		db.EXPECT().BeginTransaction(uint32(0)),
		db.EXPECT().GetBalance(testAddr).Return(balance),
		log.EXPECT().Debug(getBalance),
		log.EXPECT().Debug(endTransaction),
		db.EXPECT().EndTransaction(),
		log.EXPECT().Debug(endBlock),
		db.EXPECT().EndBlock(),
	)

	ctx.State.BeginBlock(1)
	ctx.State.BeginTransaction(0)
	ctx.State.GetBalance(testAddr)
	ctx.State.EndTransaction()
	ctx.State.EndBlock()

	err = ext.PostRun(executor.State[any]{}, ctx, nil)
	if err != nil {
		t.Fatalf("post-run returned err; %v", err)
	}

	// signal and await the close
	close(ext.input)
	ext.wg.Wait()

	file, err := os.Open(fileName)
	if err != nil {
		t.Fatalf("cannot open testing; %v", err)
	}
	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("cannot read content of the testing log; %v", err)
	}

	got := strings.TrimSpace(string(fileContent))
	want := strings.TrimSpace("BeginBlock, 1\nBeginTransaction, 0\nGetBalance, 0x0000000000000000000000000000000000000000, 10\nEndTransaction\nEndBlock")

	if strings.Compare(got, want) != 0 {
		t.Fatalf("unexpected file output\nGot: %v\nWant: %v", got, want)
	}
}

func TestDbLoggerExtension_PreTransactionCreatesNewLoggerProxy(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	cfg := &utils.Config{}
	cfg.DbLogging = t.TempDir() + "test-log"
	cfg.LogLevel = "critical"

	ctx := new(executor.Context)
	ctx.State = db

	ext := MakeDbLogger[any](cfg)

	// ctx.State is not yet a LoggerProxy hence PreTransaction assigns it
	err := ext.PreTransaction(executor.State[any]{}, ctx)
	if err != nil {
		t.Fatalf("pre-transaction failed; %v", err)
	}

	if _, ok := ctx.State.(*proxy.LoggingStateDb); !ok {
		t.Fatal("db must be of type LoggingStateDb!")
	}
}

func TestDbLoggerExtension_PreTransactionDoesNotCreateNewLoggerProxy(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	cfg := &utils.Config{}
	cfg.DbLogging = t.TempDir() + "test-log"
	cfg.LogLevel = "critical"

	ctx := new(executor.Context)
	ctx.State = db

	ext := MakeDbLogger[any](cfg)

	// first call PreTransaction to assign the proxy
	err := ext.PreTransaction(executor.State[any]{}, ctx)
	if err != nil {
		t.Fatalf("pre-transaction failed; %v", err)
	}

	// save original state to make sure next call to PreTransaction will not have changed the ctx.State
	originalDb := ctx.State

	// then make sure it is not re-assigned again
	err = ext.PreTransaction(executor.State[any]{}, ctx)
	if err != nil {
		t.Fatalf("pre-transaction failed; %v", err)
	}

	if originalDb != ctx.State {
		t.Fatal("db must not be be changed!")
	}
}

func TestDbLoggerExtension_PreRunCreatesNewLoggerProxyIfStateIsNotNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	cfg := &utils.Config{}
	cfg.DbLogging = t.TempDir() + "test-log"
	cfg.LogLevel = "critical"

	ctx := new(executor.Context)
	ctx.State = db

	ext := MakeDbLogger[any](cfg)

	err := ext.PreRun(executor.State[any]{}, ctx)
	if err != nil {
		t.Fatalf("pre-transaction failed; %v", err)
	}

	if _, ok := ctx.State.(*proxy.LoggingStateDb); !ok {
		t.Fatal("db must be of type LoggingStateDb!")
	}
}

func TestDbLoggerExtension_PreRunDoesNotCreateNewLoggerProxyIfStateIsNil(t *testing.T) {
	cfg := &utils.Config{}
	cfg.DbLogging = t.TempDir() + "test-log"
	cfg.LogLevel = "critical"

	ctx := new(executor.Context)

	ext := MakeDbLogger[any](cfg)

	err := ext.PreRun(executor.State[any]{}, ctx)
	if err != nil {
		t.Fatalf("pre-transaction failed; %v", err)
	}

	if ctx.State != nil {
		t.Fatal("db must be nil!")
	}
}

func TestDbLoggerExtension_StateDbCloseIsWrittenInTheFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	fileName := t.TempDir() + "test-log"
	cfg := &utils.Config{}
	cfg.DbLogging = fileName

	ext := makeDbLogger[any](cfg, log)

	ctx := &executor.Context{State: db}

	err := ext.PreRun(executor.State[any]{}, ctx)
	if err != nil {
		t.Fatalf("pre-run returned err; %v", err)
	}

	err = ext.PreTransaction(executor.State[any]{}, ctx)
	if err != nil {
		t.Fatalf("pre-transaction returned err; %v", err)
	}

	want := "Close"
	gomock.InOrder(
		db.EXPECT().Close().Return(nil),
		log.EXPECT().Debug(want),
	)

	err = ctx.State.Close()
	if err != nil {
		t.Fatalf("cannot close database; %v", err)
	}

	file, err := os.Open(fileName)
	if err != nil {
		t.Fatalf("cannot open testing; %v", err)
	}
	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("cannot read content of the testing log; %v", err)
	}

	if !strings.Contains(string(fileContent), want) {
		t.Fatalf("close was not logged\nlog: %v", string(fileContent))
	}
}

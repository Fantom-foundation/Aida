package logger

import (
	"fmt"
	"io"
	"math/big"
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

	balance := new(big.Int).SetInt64(10)

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

	stat, err := os.Stat(fileName)
	if err != nil {
		t.Fatalf("cannot get file stats; %v", err)
	}

	if stat.Size() == 0 {
		t.Fatal("log file should have something inside")
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

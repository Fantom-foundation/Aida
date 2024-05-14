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
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestErrorLogger_FileIsNotCreatedIfNotDefined(t *testing.T) {
	cfg := &utils.Config{}
	ext := makeErrorLogger[any](cfg, logger.NewLogger("critical", "Test"))
	ext.PreRun(executor.State[any]{}, new(executor.Context))

	if ext.file != nil {
		t.Error("file must be nil")
	}
}

func TestErrorLogger_PostRunClosesLoggingThreadAndDoesNotBlockTheExecution(t *testing.T) {
	cfg := &utils.Config{}
	ext := makeErrorLogger[any](cfg, logger.NewLogger("critical", "Test"))

	ctx := new(executor.Context)

	ext.PreRun(executor.State[any]{}, ctx)

	// make sure PostRun is not blocking.
	done := make(chan bool)
	go func() {
		if err := ext.PostRun(executor.State[any]{}, ctx, nil); err != nil {
			t.Errorf("unexpected error; %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(time.Second):
		t.Fatalf("PostRun blocked unexpectedly")
	}
}

func TestErrorLogger_LoggingHappens(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	fileName := t.TempDir() + "test-log"
	cfg := &utils.Config{}
	cfg.ContinueOnFailure = true
	cfg.ErrorLogging = fileName
	ext := makeErrorLogger[any](cfg, log)

	e := errors.New("testing error")

	gomock.InOrder(
		log.EXPECT().Noticef(gomock.Any(), gomock.Any()),
		log.EXPECT().Errorf("New error: \n\t%v", e),
		log.EXPECT().Warningf("Total number of errors %v", 1),
	)

	ctx := new(executor.Context)

	err := ext.PreRun(executor.State[any]{}, ctx)
	if err != nil {
		t.Fatalf("post-run returned err")
	}

	ctx.ErrorInput <- e

	err = ext.PostRun(executor.State[any]{}, ctx, nil)
	if err == nil {
		t.Fatalf("post-run must return err")
	}

	expectedErr := fmt.Sprintf("total 1 errors occurred: %v", e)
	got := err.Error()

	if strings.Compare(got, expectedErr) != 0 {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", expectedErr, got)
	}

	stat, err := os.Stat(fileName)
	if err != nil {
		t.Fatalf("cannot get file stats; %v", err)
	}

	if stat.Size() == 0 {
		t.Fatal("log file should have something inside")
	}

}

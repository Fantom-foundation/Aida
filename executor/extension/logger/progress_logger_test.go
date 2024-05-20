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
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/substate"
	"go.uber.org/mock/gomock"
)

const testProgressReportFrequency = time.Second

func TestProgressLoggerExtension_CorrectClose(t *testing.T) {
	cfg := &utils.Config{}
	ext := MakeProgressLogger[any](cfg, testProgressReportFrequency)

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

func TestProgressLoggerExtension_NoLoggerIsCreatedIfDisabled(t *testing.T) {
	cfg := &utils.Config{}
	cfg.NoHeartbeatLogging = true
	ext := MakeProgressLogger[any](cfg, testProgressReportFrequency)
	if _, ok := ext.(extension.NilExtension[any]); !ok {
		t.Errorf("Logger is enabled although not set in configuration")
	}

}

func TestProgressLoggerExtension_LoggingHappens(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}

	ext := makeProgressLogger[*substate.Substate](cfg, testProgressReportFrequency, log)

	ext.PreRun(executor.State[*substate.Substate]{}, nil)

	gomock.InOrder(
		// scheduled logging
		log.EXPECT().Infof(progressLoggerReportFormat,
			gomock.Any(), 1,
			executor.MatchRate(gomock.All(executor.Gt(0.9), executor.Lt(1.1)), "txRate"),
			executor.MatchRate(gomock.All(executor.Gt(90), executor.Lt(100)), "gasRate"),
		),
		// defer logging
		log.EXPECT().Noticef(finalSummaryProgressReportFormat,
			gomock.Any(), 1,
			executor.MatchRate(gomock.All(executor.Gt(0.6), executor.Lt(0.7)), "txRate"),
			executor.MatchRate(gomock.All(executor.Gt(60), executor.Lt(70)), "gasRate"),
		),
	)

	// fill the logger with some data
	ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        nil,
	}, &executor.Context{
		ExecutionResult: substatecontext.NewReceipt(&substate.Result{GasUsed: 100_000_000}),
	})

	// we must wait for the ticker to tick
	time.Sleep((3 * testProgressReportFrequency) / 2)

	ext.PostRun(executor.State[*substate.Substate]{}, nil, nil)
}

func TestProgressLoggerExtension_LoggingHappensEvenWhenProgramEndsBeforeTickerTicks(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}

	// we set large tick rate that does not trigger the ticker
	ext := makeProgressLogger[*substate.Substate](cfg, 10*time.Second, log)

	ext.PreRun(executor.State[*substate.Substate]{}, nil)

	log.EXPECT().Noticef(finalSummaryProgressReportFormat,
		gomock.Any(), 1,
		executor.MatchRate(gomock.All(executor.Gt(0.6), executor.Lt(0.7)), "txRate"),
		executor.MatchRate(gomock.All(executor.Gt(60), executor.Lt(70)), "gasRate"),
	)

	// fill the logger with some data
	ext.PostTransaction(executor.State[*substate.Substate]{
		Block:       1,
		Transaction: 1,
		Data:        nil,
	}, &executor.Context{
		ExecutionResult: substatecontext.NewReceipt(&substate.Result{GasUsed: 100_000_000}),
	})

	// wait for data to get into logger
	time.Sleep((3 * testProgressReportFrequency) / 2)

	ext.PostRun(executor.State[*substate.Substate]{}, nil, nil)
}

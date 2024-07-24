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

package tracker

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/utils"
)

const rpcProgressTrackerReportFormat = "Track: request %d, interval_total_req_rate %.2f, interval_gas_rate %.2f, overall_total_req_rate %.2f, overall_gas_rate %.2f"

// MakeRequestProgressTracker creates a blockProgressTracker that depends on the
// PostBlock event and is only useful as part of a sequential evaluation.
func MakeRequestProgressTracker(cfg *utils.Config, reportFrequency int) executor.Extension[*rpc.RequestAndResults] {
	if !cfg.TrackProgress {
		return extension.NilExtension[*rpc.RequestAndResults]{}
	}

	if reportFrequency == 0 {
		reportFrequency = ProgressTrackerDefaultReportFrequency
	}

	return makeRequestProgressTracker(cfg, reportFrequency, logger.NewLogger(cfg.LogLevel, "ProgressTracker"))
}

func makeRequestProgressTracker(cfg *utils.Config, reportFrequency int, log logger.Logger) *requestProgressTracker {
	return &requestProgressTracker{
		progressTracker: newProgressTracker[*rpc.RequestAndResults](cfg, reportFrequency, log),
	}
}

// requestProgressTracker logs progress every XXX blocks depending on reportFrequency.
// Default is 100_000 blocks. This is mainly used for gathering information about process.
type requestProgressTracker struct {
	*progressTracker[*rpc.RequestAndResults]
	lastReportedRequestCount uint64
	overallInfo              rpcProcessInfo
	lastIntervalInfo         rpcProcessInfo
}

type rpcProcessInfo struct {
	numRequests uint64
	gas         uint64
}

// PostTransaction increments number of transactions and saves gas used in last substate.
func (t *requestProgressTracker) PostTransaction(state executor.State[*rpc.RequestAndResults], ctx *executor.Context) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.overallInfo.numRequests++
	if ctx.ExecutionResult != nil {
		t.overallInfo.gas += ctx.ExecutionResult.GetGasUsed()
	}
	overallInfo := t.overallInfo

	overallCount := overallInfo.numRequests
	if overallCount-t.lastReportedRequestCount < uint64(t.reportFrequency) {
		return nil
	}

	boundary := overallCount - (overallCount % uint64(t.reportFrequency))

	now := time.Now()
	overall := now.Sub(t.startOfRun)
	interval := now.Sub(t.startOfLastInterval)

	overallGas := overallInfo.gas
	intervalGas := t.lastIntervalInfo.gas

	intervalReqRate := float64(t.reportFrequency) / interval.Seconds()
	intervalGasRate := float64(overallGas-intervalGas) / interval.Seconds()

	overallReqRate := float64(overallCount) / overall.Seconds()
	overallGasRate := float64(overallGas) / overall.Seconds()

	t.log.Noticef(
		rpcProgressTrackerReportFormat, boundary,
		intervalReqRate, intervalGasRate,
		overallReqRate, overallGasRate,
	)

	t.lastIntervalInfo = overallInfo

	t.lastReportedRequestCount = boundary
	t.startOfLastInterval = now

	return nil
}

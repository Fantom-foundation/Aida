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

// MakeRpcProgressTracker creates a substateProgressTracker that depends on the
// PostBlock event and is only useful as part of a sequential evaluation.
func MakeRpcProgressTracker(cfg *utils.Config, reportFrequency int) executor.Extension[*rpc.RequestAndResults] {
	if !cfg.TrackProgress {
		return extension.NilExtension[*rpc.RequestAndResults]{}
	}

	if reportFrequency == 0 {
		reportFrequency = ProgressTrackerDefaultReportFrequency
	}

	return makeRpcProgressTracker(cfg, reportFrequency, logger.NewLogger(cfg.LogLevel, "ProgressTracker"))
}

func makeRpcProgressTracker(cfg *utils.Config, reportFrequency int, log logger.Logger) *rpcProgressTracker {
	return &rpcProgressTracker{
		progressTracker: newProgressTracker[*rpc.RequestAndResults](cfg, reportFrequency, log),
	}
}

// substateProgressTracker logs progress every XXX blocks depending on reportFrequency.
// Default is 100_000 blocks. This is mainly used for gathering information about process.
type rpcProgressTracker struct {
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
func (t *rpcProgressTracker) PostTransaction(state executor.State[*rpc.RequestAndResults], _ *executor.Context) error {
	// getLogs is not yet implemented - this causes nil result.
	// todo it should get implemented in upcoming PR
	if state.Data.StateDB == nil {
		return nil
	}

	// add data and get snapshot of it as quickly as possible
	t.lock.Lock()
	t.overallInfo.numRequests++
	t.overallInfo.gas += state.Data.StateDB.GasUsed
	overallInfo := t.overallInfo
	t.lock.Unlock()

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

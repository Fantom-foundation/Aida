package register

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"strconv"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/utils"
)

const (
	RegisterRequestProgressCreateTableIfNotExist = `
		CREATE TABLE IF NOT EXISTS stats_rpc (
			count INTEGER NOT NULL,
			req_rate float,
			gas_rate float,
			overall_req_rate float,
			overall_gas_rate float
		)
	`
	RegisterRequestProgressInsertOrReplace = `
		INSERT or REPLACE INTO stats_rpc (
			count,
			req_rate, gas_rate, overall_req_rate, overall_gas_rate
		) VALUES (
			?,
			?, ?, ?, ?
		)
	`
)

// MakeRegisterRequestProgress creates a blockProgressTracker that depends on the
// PostBlock event and is only useful as part of a sequential evaluation.
func MakeRegisterRequestProgress(cfg *utils.Config, reportFrequency int) executor.Extension[*rpc.RequestAndResults] {
	// As temporary measure: issue a warning to user if both RegisterRun and TrackProgress is on.
	log := logger.NewLogger(cfg.LogLevel, "RegisterRequestProgress")
	if cfg.RegisterRun != "" && cfg.TrackProgress {
		log.Warningf("Both register-run and track-progress flags are on. Both extensions uses mutexes that will result in unneccesary performance penalty. Consider using one or the other but not both.")
	}

	if cfg.RegisterRun == "" {
		return extension.NilExtension[*rpc.RequestAndResults]{}
	}

	if reportFrequency == 0 {
		reportFrequency = RegisterProgressDefaultReportFrequency
	}

	return makeRegisterRequestProgress(cfg, reportFrequency, log)
}

func makeRegisterRequestProgress(cfg *utils.Config, reportFrequency int, log logger.Logger) *registerRequestProgress {

	return &registerRequestProgress{
		cfg:             cfg,
		log:             log,
		reportFrequency: reportFrequency,
		ps:              utils.NewPrinters(),
		id:              MakeRunIdentity(time.Now().Unix(), cfg),
	}
}

// registerRequestProgress logs progress every XXX requests depending on reportFrequency.
// Default is 100_000 requests. This is mainly used for gathering information about process.
type registerRequestProgress struct {
	extension.NilExtension[*rpc.RequestAndResults]

	cfg  *utils.Config
	log  logger.Logger
	lock sync.Mutex
	ps   *utils.Printers

	// Where am I?
	lastReportedRequestCount uint64
	overallInfo              rpcProcessInfo
	lastIntervalInfo         rpcProcessInfo

	// Stats
	reportFrequency int
	startOfRun      time.Time
	lastUpdate      time.Time
	boundary        int
	intervalReqRate float64
	intervalGasRate float64
	overallReqRate  float64
	overallGasRate  float64

	id   *RunIdentity
	meta *RunMetadata
}

type rpcProcessInfo struct {
	numRequests uint64
	gas         uint64
}

func (rp *registerRequestProgress) PreRun(executor.State[*rpc.RequestAndResults], *executor.Context) error {
	connection := filepath.Join(rp.cfg.RegisterRun, fmt.Sprintf("%s.db", rp.id.GetId()))
	rp.log.Noticef("Registering to: %s", connection)

	// 1. if directory does not exists -> fatal, throw error
	if _, err := os.Stat(rp.cfg.RegisterRun); err != nil {
		return err
	}

	// 2. if database could be created -> fatal, throw error
	p2db, err := utils.NewPrinterToSqlite3(rp.sqlite3(connection))
	if err != nil {
		return err
	}
	rp.ps.AddPrinter(p2db)

	// 3. if metadata could be fetched -> continue without the failed metadata
	rm, err := MakeRunMetadata(connection, rp.id)

	// if this were to happened, it should happen already at 2 but added again just in case
	if rm == nil {
		return err
	}
	if err != nil {
		rp.log.Errorf("Metadata warning: %s.", err)
	}
	rp.meta = rm
	rp.meta.Print()

	now := time.Now()
	rp.startOfRun = now
	rp.lastUpdate = now
	return nil
}

// PostTransaction increments number of transactions and saves gas used in last substate.
func (rp *registerRequestProgress) PostTransaction(state executor.State[*rpc.RequestAndResults], _ *executor.Context) error {

	rp.lock.Lock()
	defer rp.lock.Unlock()
	
	rp.overallInfo.numRequests++
	rp.overallInfo.gas += state.Data.StateDB.GasUsed

	overallInfo := rp.overallInfo
	overallCount := overallInfo.numRequests

	if overallCount-rp.lastReportedRequestCount < uint64(rp.reportFrequency) {
		return nil
	}

	boundary := overallCount - (overallCount % uint64(rp.reportFrequency))
	rp.boundary = int(boundary)

	now := time.Now()
	sinceStartOfRun := now.Sub(rp.startOfRun)
	sinceLastUpdate := now.Sub(rp.lastUpdate)

	overallGas := overallInfo.gas
	intervalGas := rp.lastIntervalInfo.gas

	rp.intervalReqRate = float64(rp.reportFrequency) / sinceLastUpdate.Seconds()
	rp.intervalGasRate = float64(overallGas-intervalGas) / sinceLastUpdate.Seconds()

	rp.overallReqRate = float64(overallCount) / sinceStartOfRun.Seconds()
	rp.overallGasRate = float64(overallGas) / sinceStartOfRun.Seconds()

	rp.ps.Print()

	rp.lastIntervalInfo = overallInfo
	rp.lastReportedRequestCount = boundary
	rp.lastUpdate = now

	return nil
}

// PostRun prints the remaining statistics and terminates any printer resources.
func (rp *registerRequestProgress) PostRun(_ executor.State[*rpc.RequestAndResults], ctx *executor.Context, err error) error {
	rp.ps.Print()
	rp.ps.Close()

	rp.meta.meta["Runtime"] = strconv.Itoa(int(time.Since(rp.startOfRun).Seconds()))
	if err != nil {
		rp.meta.meta["RunSucceed"] = strconv.FormatBool(false)
		rp.meta.meta["RunError"] = fmt.Sprintf("%v", err)
	} else {
		rp.meta.meta["RunSucceed"] = strconv.FormatBool(true)
	}

	rp.meta.Print()
	rp.meta.Close()

	return nil
}

func (rp *registerRequestProgress) sqlite3(conn string) (string, string, string, func() [][]any) {
	return conn,
		RegisterRequestProgressCreateTableIfNotExist,
		RegisterRequestProgressInsertOrReplace,
		func() [][]any {
			return [][]any{
				{
					rp.boundary,
					rp.intervalReqRate,
					rp.intervalGasRate,
					rp.overallReqRate,
					rp.overallGasRate,
				},
			}
		}
}

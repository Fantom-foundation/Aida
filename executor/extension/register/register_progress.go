package register

import (
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
)

const (
	RegisterProgress_DefaultReportFrequency = 100_000 // in blocks

	RegisterProgress_CreateTableIfNotExist = `
		CREATE TABLE IF NOT EXISTS stats (
  			start INTEGER NOT NULL,
	  		end INTEGER NOT NULL,
			memory int,
			disk int,
	  		tx_rate float,
			gas_rate float,
  			overall_tx_rate float,
  			overall_gas_rate float,
		)
	`
	RegisterProgress_InsertOrReplace = `
		INSERT or REPLACE INTO stats (
			start, end, memory, disk, tx_rate, gas_rate, overall_tx_rate, overall_gas_rate
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?
		)
	`
)

// MakeRegisterProgress creates an extention that
//  1. Track Progress e.g. ProgressTracker
//  2. Register the intermediate results to an external service (sqlite3 db)
func MakeRegisterProgress(cfg *utils.Config, reportFrequency int) executor.Extension[*substate.Substate] {
	if cfg.RegisterRun == "" {
		return extension.NilExtension[*substate.Substate]{}
	}

	if reportFrequency == 0 {
		reportFrequency = RegisterProgress_DefaultReportFrequency
	}

	rp := &registerProgress{
		cfg:      cfg,
		log:      logger.NewLogger(cfg.LogLevel, "Register-Progress-Logger"),
		interval: utils.NewInterval(cfg.First, cfg.Last, uint64(reportFrequency)),
		ps:       utils.NewPrinters(),
	}

	p2db, err := utils.NewPrinterToSqlite3(rp.sqlite3(cfg.RegisterRun))

	if err != nil {
		rp.log.Debugf("Unable to register at %s", cfg.RegisterRun)
	} else {
		rp.ps.AddPrinter(p2db)
	}

	return rp
}

// registerProgress logs progress every XXX blocks depending on reportFrequency.
// Default is 100_000 blocks. This is mainly used for gathering information about process.
type registerProgress struct {
	extension.NilExtension[*substate.Substate]

	// Configuration
	cfg  *utils.Config
	log  logger.Logger
	lock sync.Mutex
	ps   *utils.Printers

	// Where am I?
	interval           *utils.Interval
	lastProcessedBlock int

	// Stats
	startOfRun   time.Time
	lastUpdate   time.Time
	txCount      uint64
	gas          uint64
	totalTxCount uint64
	totalGas     uint64
	directory    string
	memory       *state.MemoryUsage
}

func (rp *registerProgress) PreRun(_ executor.State[*substate.Substate], _ *executor.Context) error {
	now := time.Now()
	rp.startOfRun = now
	rp.lastUpdate = now
	rp.directory = ctx.StateDbPath
	return nil
}

// PreBlock sends the state to the report goroutine.
// We only care about total number of transactions we can do this here rather in Pre/PostTransaction.
//
// This is done in PreBlock because some blocks do not have transaction.
func (rp *registerProgress) PreBlock(state executor.State[*substate.Substate], ctx *executor.Context) error {
	if uint64(state.Block) > rp.interval.End() {
		rp.memory = ctx.State.GetMemoryUsage()
		rp.ps.Print()
		rp.Reset()
		rp.interval.Next()
	}

	return nil
}

// PostTransaction increments number of transactions and saves gas used in last substate.
func (rp *registerProgress) PostTransaction(state executor.State[*substate.Substate], _ *executor.Context) error {
	rp.lock.Lock()
	defer rp.lock.Unlock()

	rp.totalTxCount++
	rp.txCount++

	rp.totalGas += state.Data.Result.GasUsed
	rp.gas += state.Data.Result.GasUsed

	return nil
}

// PostRun prints the remaining statistics and terminates any printer resources.
func (rp *registerProgress) PostRun(_ executor.State[*substate.Substate], ctx *executor.Context, _ error) error {
	rp.memory = ctx.State.GetMemoryUsage()
	rp.ps.Print()
	rp.Reset()
	rp.ps.Close()

	return nil
}

// Reset set local interval trackers to initial state for the next interval.
func (rp *registerProgress) Reset() {
	rp.lastUpdate = time.Now()
	rp.txCount = 0
	rp.gas = 0
}

func (rp *registerProgress) sqlite3(conn string) (string, string, string, func() [][]any) {
	return conn, RegisterProgress_CreateTableIfNotExist, RegisterProgress_InsertOrReplace,
		func() [][]any {
			values := [][]any{}

			var (
				txCount      uint64
				gas          uint64
				totalTxCount uint64
				totalGas     uint64
			)

			rp.lock.Lock()
			txCount = rp.txCount
			gas = rp.gas
			totalTxCount = rp.totalTxCount
			totalGas = rp.totalGas
			rp.lock.Unlock()

			disk, _ := utils.GetDirectorySize(rp.directory)
			mem := rp.memory.UsedBytes

			txRate := float64(txCount) / time.Since(rp.lastUpdate).Seconds()
			gasRate := float64(gas) / time.Since(rp.lastUpdate).Seconds()
			overallTxRate := float64(totalTxCount) / time.Since(rp.startOfRun).Seconds()
			overallGasRate := float64(totalGas) / time.Since(rp.startOfRun).Seconds()

			values = append(values, []any{
				rp.interval.Start(),
				rp.interval.End(),
				disk,
				mem,
				txRate,
				gasRate,
				overallTxRate,
				overallGasRate,
			})

			return values
		}
}


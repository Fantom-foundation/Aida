package register

import (
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

const (
	RegisterProgressDefaultReportFrequency = 100_000 // in blocks

	RegisterProgressCreateTableIfNotExist = `
		CREATE TABLE IF NOT EXISTS stats (
  			start INTEGER NOT NULL,
	  		end INTEGER NOT NULL,
			memory int,
			disk int,
	  		tx_rate float,
			gas_rate float,
  			overall_tx_rate float,
  			overall_gas_rate float
		)
	`
	RegisterProgressInsertOrReplace = `
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
func MakeRegisterProgress(cfg *utils.Config, reportFrequency int) executor.Extension[txcontext.WithValidation] {
	if cfg.RegisterRun == "" {
		return extension.NilExtension[txcontext.WithValidation]{}
	}

	if reportFrequency == 0 {
		reportFrequency = RegisterProgressDefaultReportFrequency
	}

	rp := &registerProgress{
		cfg:      cfg,
		log:      logger.NewLogger(cfg.LogLevel, "Register-Progress-Logger"),
		interval: utils.NewInterval(cfg.First, cfg.Last, uint64(reportFrequency)),
		ps:       utils.NewPrinters(),
		id:       MakeRunIdentity(time.Now().Unix(), cfg),
	}

	connection := filepath.Join(cfg.RegisterRun, fmt.Sprintf("%s.db", rp.GetId()))
	rp.log.Noticef(connection)

	p2db, err := utils.NewPrinterToSqlite3(rp.sqlite3(connection))
	if err != nil {
		rp.log.Errorf("Unable to register at %s", cfg.RegisterRun)
	} else {
		rp.ps.AddPrinter(p2db)
	}

	rm, err := MakeRunMetadata(connection, rp.id)
	if err != nil {
		rp.log.Errorf("Unable to create run metadata because %s.", err)
	} else {
		rp.meta = rm
		rm.Print()
	}

	return rp
}

// registerProgress logs progress every XXX blocks depending on reportFrequency.
// Default is 100_000 blocks. This is mainly used for gathering information about process.
type registerProgress struct {
	extension.NilExtension[txcontext.WithValidation]

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

	id   *RunIdentity
	meta *RunMetadata
}

func (rp *registerProgress) PreRun(_ executor.State[txcontext.WithValidation], ctx *executor.Context) error {
	now := time.Now()
	rp.startOfRun = now
	rp.lastUpdate = now
	rp.directory = ctx.StateDbPath
	return nil
}

// PreBlock sends the state to the report goroutine.
// We only care about total number of transactions we can do this here rather in Pre/PostTransaction.
//
// This is done in PreBlock because some blocks do not have txcontext.
func (rp *registerProgress) PreBlock(state executor.State[txcontext.WithValidation], ctx *executor.Context) error {
	if uint64(state.Block) > rp.interval.End() {
		rp.memory = ctx.State.GetMemoryUsage()
		rp.ps.Print()
		rp.Reset()
		rp.interval.Next()
	}

	return nil
}

// PostTransaction increments number of transactions and saves gas used in last substate.
func (rp *registerProgress) PostTransaction(state executor.State[txcontext.WithValidation], _ *executor.Context) error {
	res := state.Data.GetReceipt()

	rp.lock.Lock()
	defer rp.lock.Unlock()

	rp.totalTxCount++
	rp.txCount++

	rp.totalGas += res.GetGasUsed()
	rp.gas += res.GetGasUsed()

	return nil
}

// PostRun prints the remaining statistics and terminates any printer resources.
func (rp *registerProgress) PostRun(_ executor.State[txcontext.WithValidation], ctx *executor.Context, err error) error {
	rp.memory = ctx.State.GetMemoryUsage()
	rp.ps.Print()
	rp.Reset()
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

// Reset set local interval trackers to initial state for the next interval.
func (rp *registerProgress) Reset() {
	rp.lastUpdate = time.Now()
	rp.txCount = 0
	rp.gas = 0
}

// GetId returns a unique id based on the run metadata.
func (rp *registerProgress) GetId() string {
	return rp.id.GetId()
}

func (rp *registerProgress) sqlite3(conn string) (string, string, string, func() [][]any) {
	return conn, RegisterProgressCreateTableIfNotExist, RegisterProgressInsertOrReplace,
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

			disk, err := utils.GetDirectorySize(rp.directory)
			if err != nil {
				rp.log.Errorf("Unable to get directory size from %s", rp.directory)
				return [][]any{}
			}

			mem := rp.memory.UsedBytes

			txRate := float64(txCount) / time.Since(rp.lastUpdate).Seconds()
			gasRate := float64(gas) / time.Since(rp.lastUpdate).Seconds()
			overallTxRate := float64(totalTxCount) / time.Since(rp.startOfRun).Seconds()
			overallGasRate := float64(totalGas) / time.Since(rp.startOfRun).Seconds()

			values = append(values, []any{
				rp.interval.Start(),
				rp.interval.End(),
				mem,
				disk,
				txRate,
				gasRate,
				overallTxRate,
				overallGasRate,
			})

			return values
		}
}

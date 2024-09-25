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

package register

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	rr "github.com/Fantom-foundation/Aida/register"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

type whenToPrint int

const (
	OnPreBlock whenToPrint = iota
	OnPreTransaction
)

const (
	archiveDbDirectoryName = "archive"
	defaultReportFrequency uint64 = 100_000

	registerProgressCreateTableIfNotExist = `
		CREATE TABLE IF NOT EXISTS stats (
  			start INTEGER NOT NULL,
	  		end INTEGER NOT NULL,
			memory int,
			live_disk int,
			archive_disk int,
	  		tx_rate float,
			gas_rate float,
  			overall_tx_rate float,
  			overall_gas_rate float
		)
	`
	registerProgressInsertOrReplace = `
		INSERT or REPLACE INTO stats (
			start, end, 
			memory, live_disk, archive_disk, 
			tx_rate, gas_rate, overall_tx_rate, overall_gas_rate
		) VALUES (
			?, ?, 
			?, ?, ?, 
			?, ?, ?, ?
		)
	`
)

// MakeRegisterProgress creates an extention that
//  1. Track Progress e.g. ProgressTracker
//  2. Register the intermediate results to an external service (sqlite3 db)
func MakeRegisterProgress(cfg *utils.Config, reportFrequency int, when whenToPrint) executor.Extension[txcontext.TxContext] {
	if cfg.RegisterRun == "" {
		return extension.NilExtension[txcontext.TxContext]{}
	}

	var freq uint64 = defaultReportFrequency
	if reportFrequency > 0 {
		freq = uint64(reportFrequency)
	}

	return &registerProgress{
		cfg:      cfg,
		log:      logger.NewLogger(cfg.LogLevel, "Register-Progress-Logger"),
		interval: utils.NewInterval(cfg.First, cfg.Last, freq),
		when:     when,
		ps:       utils.NewPrinters(),
		id:       rr.MakeRunIdentity(time.Now().Unix(), cfg),
	}
}

// registerProgress logs progress every XXX blocks depending on reportFrequency.
// Default is 100_000 blocks. This is mainly used for gathering information about process.
type registerProgress struct {
	extension.NilExtension[txcontext.TxContext]

	// Configuration
	cfg  *utils.Config
	log  logger.Logger
	lock sync.Mutex
	ps   *utils.Printers
	when whenToPrint

	// Where am I?
	interval           *utils.Interval
	lastProcessedBlock int

	// Stats
	startOfRun      time.Time
	lastUpdate      time.Time
	txCount         uint64
	gas             uint64
	totalTxCount    uint64
	totalGas        uint64
	pathToStateDb   string
	pathToArchiveDb string
	memory          *state.MemoryUsage

	id   *rr.RunIdentity
	meta *rr.RunMetadata
}

// PreRun checks the following items:
// 1. if directory does not exists -> fatal, throw error
// 2. if database could not be created -> fatal, throw error
// 3. if metadata table could not be created -> fatal, throw error
func (rp *registerProgress) PreRun(_ executor.State[txcontext.TxContext], ctx *executor.Context) error {
	connection := filepath.Join(rp.cfg.RegisterRun, fmt.Sprintf("%s.db", rp.GetId()))
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
	rm, err := rr.MakeRunMetadata(connection, rp.id, rr.FetchUnixInfo)

	// if this were to happened, it should happen already at 2 but added again just in case
	if rm == nil {
		return err
	}
	if err != nil {
		rp.log.Errorf("Metadata warnings: %s.", err)
	}
	rp.meta = rm
	rm.Print()

	// Proceed
	now := time.Now()
	rp.startOfRun = now
	rp.lastUpdate = now
	rp.pathToStateDb = ctx.StateDbPath
	if strings.ToLower(rp.cfg.DbImpl) == "carmen" {
		rp.pathToArchiveDb = filepath.Join(ctx.StateDbPath, archiveDbDirectoryName)
	} else {
		rp.pathToArchiveDb = ctx.StateDbPath
	}

	// Check if any path-to-state-db is not initialized, terminate now if so
	_, err = utils.GetDirectorySize(rp.pathToStateDb)
	if err != nil {
		rp.log.Errorf("Failed to get directory size of state db at path: %s", rp.pathToStateDb)
		return err
	}

	if rp.cfg.ArchiveMode {
		_, err = utils.GetDirectorySize(rp.pathToArchiveDb)
		if err == nil {
			rp.log.Errorf("Failed to get directory size of archive db at path: %s", rp.pathToStateDb)
			return err
		}
	}

	return nil
}

func (rp *registerProgress) PreBlock(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	if rp.when != OnPreBlock {
		return nil
	}

	if uint64(state.Block) > rp.interval.End() {
		return rp.printAndReset(ctx)
	}

	return nil
}

func (rp *registerProgress) PreTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	if rp.when != OnPreTransaction {
		return nil
	}

	if uint64(state.Block) > rp.interval.End() {
		return rp.printAndReset(ctx)
	}

	return nil
}

// printAndReset sends the state to the report goroutine and reset current-interval tracker.
func (rp *registerProgress) printAndReset(ctx *executor.Context) error {
	rp.memory = ctx.State.GetMemoryUsage()
	rp.ps.Print()
	rp.Reset()
	rp.interval.Next()

	return nil
}

// PostTransaction increments number of transactions and saves gas used in last substate.
func (rp *registerProgress) PostTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {

	res := ctx.ExecutionResult

	rp.lock.Lock()
	defer rp.lock.Unlock()

	rp.totalTxCount++
	rp.txCount++

	rp.totalGas += res.GetGasUsed()
	rp.gas += res.GetGasUsed()

	return nil
}

// PostRun prints the remaining statistics and terminates any printer resources.
func (rp *registerProgress) PostRun(_ executor.State[txcontext.TxContext], ctx *executor.Context, err error) error {
	rp.memory = ctx.State.GetMemoryUsage()
	rp.ps.Print()
	rp.Reset()
	rp.ps.Close()

	rp.meta.Meta["Runtime"] = strconv.Itoa(int(time.Since(rp.startOfRun).Seconds()))
	if err != nil {
		rp.meta.Meta["RunSucceed"] = strconv.FormatBool(false)
		rp.meta.Meta["RunError"] = fmt.Sprintf("%v", err)
	} else {
		rp.meta.Meta["RunSucceed"] = strconv.FormatBool(true)
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
	return conn,
		registerProgressCreateTableIfNotExist,
		registerProgressInsertOrReplace,
		func() [][]any {
			values := [][]any{}

			var (
				txCount      uint64
				gas          uint64
				totalTxCount uint64
				totalGas     uint64
				lDisk        int64
				aDisk        int64
			)

			rp.lock.Lock()
			txCount = rp.txCount
			gas = rp.gas
			totalTxCount = rp.totalTxCount
			totalGas = rp.totalGas
			rp.lock.Unlock()

			lDisk, err := utils.GetDirectorySize(rp.pathToStateDb)
			if err != nil {
				// silent defaults to 0 if anything happens to path at runtime
				lDisk = 0
			}

			if rp.cfg.ArchiveMode {
				aDisk, err = utils.GetDirectorySize(rp.pathToArchiveDb)
				if err != nil {
					// silent defaults to 0 if anything happens to path at runtime
					aDisk = 0
				} else {
					lDisk -= aDisk
				}
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
				lDisk,
				aDisk,
				txRate,
				gasRate,
				overallTxRate,
				overallGasRate,
			})

			return values
		}
}

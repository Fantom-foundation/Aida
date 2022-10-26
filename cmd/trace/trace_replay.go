package trace

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/tracer/state"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// Trace replay command
var TraceReplayCommand = cli.Command{
	Action:    traceReplayAction,
	Name:      "replay",
	Usage:     "executes storage trace",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&cpuProfileFlag,
		&disableProgressFlag,
		&profileFlag,
		&stateDbImplementation,
		&substate.SubstateDirFlag,
		&substate.WorkersFlag,
		&traceDirectoryFlag,
		&traceDebugFlag,
		&validateEndState,
	},
	Description: `
The trace replay command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay storage traces.`,
}

// generateUpdateDatabase generates an update set for a block range.
func generateUpdateSet(first uint64, last uint64, int numWorkers) substate.SubstateAlloc {
	stateIter := substate.NewSubstateIterator(first, numWorkers)
	defer stateIter.Release()
	var update substate.SubstateAlloc
	for stateIter.Next() {
		tx := stateIter.Value()
		// exceeded block range?
		if tx.Block > last {
			break
		}
		// merge output sub-state to update
		update.Merge(tx.Substate.OutputAlloc)
	}
	return update
}

// generateWorldState generates a world-state for a block
func  generateWorldState(block uint64, int numWorkers) substate.SubstateAlloc {
	// load initial worldstate for block 4.5M
	// load here
	// ws := loadInitWorldState()
	ws substate.SubstateAlloc = make(substate.SubstateAlloc)
	
	// generate world state for block 
	update := generateUpdateSet(45000000,last, numWorkers)
	ws.Merge(update)
	return ws
}

// prime database 
func primeDatabase(worldState substate.SubstateAlloc, db state.StateDB) {
	// TODO: Extend so that priming order is randomized
	for  addr, account := range recordedAlloc {
		db.CreateAccount(addr)	
		db.AddBalance(addr, account.Balance)
		db.SetNonce(addr, account.Nonce)
		db.SetCode(addr, account.Code)
		for key, value := account.Storage {
			db.SetState(addr, key, value)
		}
	}
}

// validate database 
// NB: We can only check what must be in the db (but cannot check 
// whether db stores more)
// Perhaps reuse some of the code from 
fund validateDatabase(worldState substate.SubstateAlloc, db state.StateDB) bool {
	// TODO: Extend so that priming order is randomized
	for  addr, account := range recordedAlloc {
		if  db.Exist(addr) {
			log.Fatalf("Account %v does not exist", addr.Hex())
		}
		if  db.GetBalance(addr) != account.GetBalance() {
			// TODO: print more detail
			log.Fatalf("Failed to validate balance for account %v", addr.Hex())
		}
		if  db.SetNonce(addr, account.Nonce) != account.GetNonce() {
			// TODO: print more detail
			log.Fatalf("Failed to validate nonce for account %v", addr.Hex())
		}
		if  db.GetCode(addr, account.Nonce) != account.GetNonce() {
			// TODO: print more detail
			log.Fatalf("Failed to validate code for account %v", addr.Hex())
		}
		// db.SetCode(addr, account.Code)
		for key, value := account.Storage {
			if db.GetState(addr, key, value) != acccount.GetState(addr, key, value) {
				// TODO: print more detail
				log.Fatalf("Failed to validate nonce for account %v", addr.Hex())
			}
		}
	}
	return true
}

// Compare state after replaying traces with recorded state.
// TODO: Perhaps retire this or move to a test-case??
func compareStorage(recordedAlloc substate.SubstateAlloc, traceAlloc substate.SubstateAlloc) error {
	for account, recordAccount := range recordedAlloc {
		// account exists in both substate
		if replayAccout, exist := traceAlloc[account]; exist {
			for k, xv := range recordAccount.Storage {
				// mismatched value or key dones't exist
				if yv, exist := replayAccout.Storage[k]; !exist || xv != yv {
					return fmt.Errorf("Error: mismatched value at storage key %v. want %v have %v\n", k, xv, yv)
				}
			}
			for k, yv := range replayAccout.Storage {
				// key exists when expecting nil
				if xv, exist := recordAccount.Storage[k]; !exist {
					return fmt.Errorf("Error: mismatched value at storage key %v. want %v have %v\n", k, xv, yv)
				}
			}
		} else {
			if len(recordAccount.Storage) > 0 {
				return fmt.Errorf("Error: account %v doesn't exist\n", account)
			}
			//else ignores accounts which has no storage
		}
	}

	// checks for unexpected accounts in replayed substate
	for account := range traceAlloc {
		if _, exist := recordedAlloc[account]; !exist {
			return fmt.Errorf("Error: unexpected account %v\n", account)
		}
	}
	return nil
}

// Create a new DB instance based on cli argument.
func makeStateDb(directory string, cliCtx *cli.Context) (state.StateDB, error) {
	impl := cliCtx.String(stateDbImplementation.Name)
	switch impl {
	case "memory":
		return state.MakeGethInMemoryStateDB(), nil
	case "geth":
		return state.MakeGethStateDB(directory)
	case "carmen":
		return state.MakeCarmenStateDB(directory)
	}
	return nil, fmt.Errorf("Unknown DB implementation (--%v): %v", stateDbImplementation.Name, impl)
}

// getDirectorySize computes the size of all files in the given directoy in bytes.
func getDirectorySize(directory string) int64 {
	var sum int64 = 0
	filepath.Walk(directory, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			sum += info.Size()
		}
		return nil
	})
	return sum
}

// Simulate storage operations from storage traces on stateDB.
func storageDriver(first uint64, last uint64, cliCtx *cli.Context) error {
	// load dictionaries & indexes
	dCtx := dict.ReadDictionaryContext()
	iCtx := tracer.ReadIndexContext()

	// TODO: 1) compute full-state for "first" block, and
	//       2) transcribe full-state to the StateDB object
	//          under test.

	// iterate substate (for in-membory state)
	// TODO set configurable number of workers
	stateIter := substate.NewSubstateIterator(first, cliCtx.Int(substate.WorkersFlag.Name))
	defer stateIter.Release()

	// replay storage trace
	traceIter := tracer.NewTraceIterator(iCtx, first, last)
	defer traceIter.Release()

	// Get validation flag
	validation_enabled := cliCtx.Bool(validateEndState.Name)

	// Get profiling flag
	operation.Profiling = cliCtx.Bool(profileFlag.Name)

	// Get progress flag
	enableProgress := !cliCtx.Bool(disableProgressFlag.Name)

	// Start CPU profiling if requested.
	if profile_file_name := cliCtx.String(cpuProfileFlag.Name); profile_file_name != "" {
		f, err := os.Create(profile_file_name)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Create a directory for the store to place all its files.
	state_directory, err := ioutil.TempDir("", "state_db_*")
	if err != nil {
		return err
	}

	// Instantiate the state DB under testing
	db, err := makeStateDb(state_directory, cliCtx)
	if err != nil {
		return err
	}

	var (
		start   time.Time
		sec     float64
		lastSec float64
	)
	if enableProgress {
		start = time.Now()
		sec = time.Since(start).Seconds()
		lastSec = time.Since(start).Seconds()
	}
	for stateIter.Next() {
		tx := stateIter.Value()
		if tx.Block > last || !iCtx.ExistsBlock(tx.Block) {
			break
		}
		db.PrepareSubstate(&tx.Substate.InputAlloc)
		for traceIter.Next() {
			op := traceIter.Value()
			operation.Execute(op, db, dCtx)
			if traceDebug {
				operation.Debug(dCtx, op)
			}

			// find end of transaction
			if op.GetOpId() == operation.EndTransactionID {
				break
			}
		}

		// Validate stateDB and OuputAlloc
		if validation_enabled {
			traceAlloc := db.GetSubstatePostAlloc()
			recordedAlloc := tx.Substate.OutputAlloc
			err := compareStorage(recordedAlloc, traceAlloc)
			if err != nil {
				return err
			}
		}
		if enableProgress {
			// report progress
			sec = time.Since(start).Seconds()
			if sec-lastSec >= 15 {
				fmt.Printf("trace replay: elasped time: %.0f s, at block %v\n", sec, tx.Block)
				lastSec = sec
			}
		}
	}

	// replay the last EndBlock()
	hasNext := traceIter.Next()
	op := traceIter.Value()
	if !hasNext || op.GetOpId() != operation.EndBlockID {
		return fmt.Errorf("Last operation isn't an EndBlock")
	} else {
		operation.Execute(op, db, dCtx)
		if traceDebug {
			operation.Debug(dCtx, op)
		}
	}

	if enableProgress {
		sec = time.Since(start).Seconds()
		fmt.Printf("trace replay: total elasped time: %.3f s, processed %v blocks\n", sec, last-first+1)
	}

	// print profile statistics (if enabled)
	if operation.Profiling {
		operation.PrintProfiling()
	}

	// close the DB and print disk usage
	start = time.Now()
	if err := db.Close(); err != nil {
		fmt.Printf("Failed to close database: %v", err)
	}
	fmt.Printf("Closing DB took %v\n", time.Since(start))
	fmt.Printf("Final disk usage: %v MiB\n", float32(getDirectorySize(state_directory))/float32(1024*1024))

	return nil
}

// Implements trace command for replaying.
func traceReplayAction(ctx *cli.Context) error {
	var err error

	// process arguments
	if ctx.Args().Len() != 2 {
		return fmt.Errorf("trace replay command requires exactly 2 arguments")
	}
	tracer.TraceDir = ctx.String(traceDirectoryFlag.Name) + "/"
	dict.DictDir = ctx.String(traceDirectoryFlag.Name) + "/"
	if ctx.Bool(traceDebugFlag.Name) {
		traceDebug = true
	}
	first, last, argErr := SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1))
	if argErr != nil {
		return argErr
	}

	// run storage driver
	substate.SetSubstateFlags(ctx)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()
	err = storageDriver(first, last, ctx)

	return err
}

package stochastic

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/stochastic/exponential"
	"github.com/Fantom-foundation/Aida/stochastic/generator"
	"github.com/Fantom-foundation/Aida/stochastic/statistics"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
)

// Simulation constants
// TODO: convert constants to CLI parameters so that they can be changed without recompiling.
const (
	AddBalanceRange = 100000  // balance range for adding value to an account
	SetNonceRange   = 1000000 // nonce range
	MaxCodeSize     = 24576   // fixed upper limit by EIP-170
	FinaliseFlag    = true    // flag for Finalise() StateDB operation
)

// stochasticState keeps the execution state for the stochastic simulation
type stochasticState struct {
	db             state.StateDB             // StateDB database
	contracts      *generator.IndirectAccess // index access generator for contracts
	keys           *generator.RandomAccess   // index access generator for keys
	values         *generator.RandomAccess   // index access generator for values
	snapshotLambda float64                   // lambda parameter for snapshot delta distribution
	totalTx        uint64                    // total number of transactions
	txNum          uint32                    // current transaction number
	blockNum       uint64                    // current block number
	syncPeriodNum  uint64                    // current sync-period number
	snapshot       []int                     // stack of active snapshots
	suicided       []int64                   // list of suicided accounts
	traceDebug     bool                      // trace-debug flag
	rg             *rand.Rand                // random generator for sampling
}

// find is a helper function to find an element in a slice
func find[T comparable](a []T, x T) int {
	for idx, y := range a {
		if x == y {
			return idx
		}
	}
	return -1
}

// createState creates a stochastic state and primes the StateDB
func createState(cfg *utils.Config, e *EstimationModelJSON, db state.StateDB, rg *rand.Rand) *stochasticState {
	// produce random access generators for contract addresses,
	// storage-keys, and storage addresses.
	// (NB: Contracts need an indirect access wrapper because
	// contract addresses can be deleted by suicide.)
	contracts := generator.NewIndirectAccess(generator.NewRandomAccess(
		rg,
		e.Contracts.NumKeys,
		e.Contracts.Lambda,
		e.Contracts.QueueDistribution,
	))
	keys := generator.NewRandomAccess(
		rg,
		e.Keys.NumKeys,
		e.Keys.Lambda,
		e.Keys.QueueDistribution,
	)
	values := generator.NewRandomAccess(
		rg,
		e.Values.NumKeys,
		e.Values.Lambda,
		e.Values.QueueDistribution,
	)

	// setup state
	ss := NewStochasticState(rg, db, contracts, keys, values, e.SnapshotLambda)

	// create accounts in StateDB
	ss.prime()

	return &ss
}

// getStochasticMatrix returns the stochastic matrix with its operations and the initial state
func getStochasticMatrix(e *EstimationModelJSON) ([]string, [][]float64, int) {
	operations := e.Operations
	A := e.StochasticMatrix
	// and set initial state to BeginSyncPeriod
	state := find(operations, OpMnemo(BeginSyncPeriodID))
	if state == -1 {
		panic("BeginSyncPeriod cannot be observed in stochastic matrix/recording failed.")
	}
	return operations, A, state
}

// retrieve operations and stochastic matrix from simulation object

// RunStochasticReplay runs the stochastic simulation for StateDB operations.
// It requires the simulation model and simulation length. The trace-debug flag
// enables/disables the printing of StateDB operations and their arguments on
// the screen.
func RunStochasticReplay(db state.StateDB, e *EstimationModelJSON, nBlocks int, cfg *utils.Config) error {
	var (
		opFrequency [NumOps]uint64 // operation frequency
		numOps      uint64         // total number of operations
	)

	// random generator
	rg := rand.New(rand.NewSource(cfg.RandomSeed))
	log.Printf("using random seed %d", cfg.RandomSeed)

	// create a stochastic state
	ss := createState(cfg, e, db, rg)

	// get stochastic matrix
	operations, A, state := getStochasticMatrix(e)

	// progress message setup
	var (
		start    time.Time
		sec      float64
		lastSec  float64
		runErr   error
		errCount int
	)

	if cfg.EnableProgress {
		start = time.Now()
		sec = time.Since(start).Seconds()
		lastSec = time.Since(start).Seconds()
	}
	// if block after priming is greater or equal to debug block, enable debug.
	if cfg.Debug && ss.blockNum >= cfg.DebugFrom {
		ss.enableDebug()
	}

	block := 0
	// inclusive range
	log.Printf("Simulation block range: first %v, last %v\n", ss.blockNum, ss.blockNum+uint64(nBlocks-1))
	for {

		// decode opcode
		op, addrCl, keyCl, valueCl := DecodeOpcode(operations[state])

		// keep track of stats
		numOps++
		opFrequency[op]++

		// execute operation with its argument classes
		ss.execute(op, addrCl, keyCl, valueCl)

		// check for end of simulation
		if op == EndBlockID {
			block++
			if block >= nBlocks {
				break
			}
			// if current block is greater or equal to debug block, enable debug.
			if cfg.Debug && !ss.traceDebug && ss.blockNum >= cfg.DebugFrom {
				ss.enableDebug()
			}
		}

		if cfg.EnableProgress {
			// report progress
			sec = time.Since(start).Seconds()
			if sec-lastSec >= 15 {
				log.Printf("Elapsed time: %.0f s, at block %v\n", sec, block)
				lastSec = sec
			}
		}

		// check for errors
		if err := ss.db.Error(); err != nil {
			errCount++
			if runErr == nil {
				runErr = fmt.Errorf("Error: stochastic replay failed.")
			}

			runErr = fmt.Errorf("%v\n\tBlock %v Tx %v: %v", runErr, ss.blockNum, ss.txNum, err)
			if !cfg.ContinueOnFailure {
				break
			}
		}

		// transit to next state in Markovian process
		state = nextState(rg, A, state)
	}

	// print progress summary
	if cfg.EnableProgress {
		log.Printf("Total elapsed time: %.3f s, processed %v blocks\n", sec, block)
	}
	if errCount > 0 {
		log.Printf("%v errors were found.\n", errCount)
	}

	// print statistics
	log.Printf("SyncPeriods: %v", ss.syncPeriodNum)
	log.Printf("Blocks: %v", ss.blockNum)
	log.Printf("Transactions: %v", ss.totalTx)
	log.Printf("Operations: %v", numOps)
	log.Printf("Operation Frequencies:")
	for op := 0; op < NumOps; op++ {
		log.Printf("\t%v: %v", opText[op], opFrequency[op])
	}
	return runErr
}

// NewStochasticState creates a new state for execution StateDB operations
func NewStochasticState(rg *rand.Rand, db state.StateDB, contracts *generator.IndirectAccess, keys *generator.RandomAccess, values *generator.RandomAccess, snapshotLambda float64) stochasticState {

	// return stochastic state
	return stochasticState{
		db:             db,
		contracts:      contracts,
		keys:           keys,
		values:         values,
		snapshotLambda: snapshotLambda,
		traceDebug:     false,
		suicided:       []int64{},
		blockNum:       1,
		syncPeriodNum:  1,
		rg:             rg,
	}
}

// prime StateDB accounts using account information
func (ss *stochasticState) prime() {
	numInitialAccounts := ss.contracts.NumElem() + 1
	log.Printf("Start priming...\n")
	log.Printf("\tinitializing %v accounts\n", numInitialAccounts)
	pt := utils.NewProgressTracker(int(numInitialAccounts))
	db := ss.db
	db.BeginSyncPeriod(0)
	db.BeginBlock(0)
	db.BeginTransaction(0)

	// initialise accounts in memory with balances greater than zero
	for i := int64(0); i <= numInitialAccounts; i++ {
		addr := toAddress(i)
		db.CreateAccount(addr)
		db.AddBalance(addr, big.NewInt(ss.rg.Int63n(AddBalanceRange)))
		pt.PrintProgress()
	}
	log.Printf("Finalizing...\n")
	db.Finalise(FinaliseFlag)
	db.EndTransaction()
	db.EndBlock()
	db.EndSyncPeriod()
	log.Printf("End priming...\n")
}

// EnableDebug set traceDebug flag to true, and enable debug message when executing an operation
func (ss *stochasticState) enableDebug() {
	ss.traceDebug = true
}

// execute StateDB operations on a stochastic state.
func (ss *stochasticState) execute(op int, addrCl int, keyCl int, valueCl int) {
	var (
		addr  common.Address
		key   common.Hash
		value common.Hash
		db    state.StateDB = ss.db
		rg    *rand.Rand    = ss.rg
	)

	// fetch indexes from index access generators
	addrIdx := ss.contracts.NextIndex(addrCl)
	keyIdx := ss.keys.NextIndex(keyCl)
	valueIdx := ss.values.NextIndex(valueCl)

	// convert index to address/hashes
	if addrCl != statistics.NoArgID {
		addr = toAddress(addrIdx)
	}
	if keyCl != statistics.NoArgID {
		key = toHash(keyIdx)
	}
	if valueCl != statistics.NoArgID {
		value = toHash(valueIdx)
	}

	// print opcode and its arguments
	if ss.traceDebug {
		// print operation
		fmt.Printf("opcode:%v (%v)", opText[op], EncodeOpcode(op, addrCl, keyCl, valueCl))

		// print indexes of contract address, storage key, and storage value.
		if addrCl != statistics.NoArgID {
			fmt.Printf(" addr-idx: %v", addrIdx)
		}
		if keyCl != statistics.NoArgID {
			fmt.Printf(" key-idx: %v", keyIdx)
		}
		if valueCl != statistics.NoArgID {
			fmt.Printf(" value-idx: %v", valueIdx)
		}
	}

	switch op {
	case AddBalanceID:
		value := rg.Int63n(AddBalanceRange)
		if ss.traceDebug {
			fmt.Printf(" value: %v", value)
		}
		db.AddBalance(addr, big.NewInt(value))

	case BeginBlockID:
		if ss.traceDebug {
			fmt.Printf(" id: %v", ss.blockNum)
		}
		db.BeginBlock(ss.blockNum)
		ss.txNum = 0
		ss.suicided = []int64{}

	case BeginSyncPeriodID:
		if ss.traceDebug {
			fmt.Printf(" id: %v", ss.syncPeriodNum)
		}
		db.BeginSyncPeriod(ss.syncPeriodNum)

	case BeginTransactionID:
		if ss.traceDebug {
			fmt.Printf(" id: %v", ss.txNum)
		}
		db.BeginTransaction(ss.txNum)
		ss.snapshot = []int{}
		ss.suicided = []int64{}

	case CreateAccountID:
		db.CreateAccount(addr)

	case EmptyID:
		db.Empty(addr)

	case EndBlockID:
		db.EndBlock()
		ss.blockNum++

	case EndSyncPeriodID:
		db.EndSyncPeriod()
		ss.syncPeriodNum++

	case EndTransactionID:
		db.EndTransaction()
		ss.txNum++
		ss.totalTx++
		ss.deleteAccounts()

	case ExistID:
		db.Exist(addr)

	case FinaliseID:
		db.Finalise(FinaliseFlag)

	case GetBalanceID:
		db.GetBalance(addr)

	case GetCodeHashID:
		db.GetCodeHash(addr)

	case GetCodeID:
		db.GetCode(addr)

	case GetCodeSizeID:
		db.GetCodeSize(addr)

	case GetCommittedStateID:
		db.GetCommittedState(addr, key)

	case GetNonceID:
		db.GetNonce(addr)

	case GetStateID:
		db.GetState(addr, key)

	case HasSuicidedID:
		db.HasSuicided(addr)

	case RevertToSnapshotID:
		snapshotNum := len(ss.snapshot)
		if snapshotNum > 0 {
			// TODO: consider a more realistic distribution
			// rather than the uniform distribution.
			snapshotIdx := snapshotNum - int(exponential.DiscreteSample(rg, ss.snapshotLambda, int64(snapshotNum))) - 1
			snapshot := ss.snapshot[snapshotIdx]
			if ss.traceDebug {
				fmt.Printf(" id: %v", snapshot)
			}
			db.RevertToSnapshot(snapshot)

			// update active snapshots and perform a rollback in balance log
			ss.snapshot = ss.snapshot[0:snapshotIdx]
		}

	case SetCodeID:
		sz := rg.Intn(MaxCodeSize-1) + 1
		if ss.traceDebug {
			fmt.Printf(" code-size: %v", sz)
		}
		code := make([]byte, sz)
		_, err := rg.Read(code)
		if err != nil {
			log.Fatalf("error producing a random byte slice. Error: %v", err)
		}
		db.SetCode(addr, code)

	case SetNonceID:
		value := uint64(rg.Intn(SetNonceRange))
		db.SetNonce(addr, value)

	case SetStateID:
		db.SetState(addr, key, value)

	case SnapshotID:
		id := db.Snapshot()
		if ss.traceDebug {
			fmt.Printf(" id: %v", id)
		}
		ss.snapshot = append(ss.snapshot, id)

	case SubBalanceID:
		balance := db.GetBalance(addr).Int64()
		if balance > 0 {
			// get a delta that does not exceed current balance
			// in the current snapshot
			value := rg.Int63n(balance)
			if ss.traceDebug {
				fmt.Printf(" value: %v", value)
			}
			db.SubBalance(addr, big.NewInt(value))
		}

	case SuicideID:
		db.Suicide(addr)
		if idx := find(ss.suicided, addrIdx); idx == -1 {
			ss.suicided = append(ss.suicided, addrIdx)
		}

	default:
		panic("invalid operation")
	}
	if ss.traceDebug {
		fmt.Println()
	}
}

// nextState produces the next state in the Markovian process.
func nextState(rg *rand.Rand, A [][]float64, i int) int {
	// Retrieve a random number in [0,1.0).
	r := rg.Float64()

	// Use Kahan's sum for summing values
	// in case we have a combination of very small
	// and very large values.
	sum := float64(0.0)
	c := float64(0.0)
	k := -1
	for j := 0; j < len(A); j++ {
		y := A[i][j] - c
		t := sum + y
		c = (t - sum) - y
		sum = t
		if r <= sum {
			return j
		}
		// If we have a numerical unstable cumulative
		// distribution (large and small numbers that cancel
		// each other out when summing up), we can take the last
		// non-zero entry as a solution. It also detects
		// stochastic matrices with a row whose row
		// sum is not zero (return value is -1 for such a case).
		if A[i][j] > 0.0 {
			k = j
		}
	}
	return k
}

// toAddress converts an address index to a contract address.
func toAddress(idx int64) common.Address {
	var a common.Address
	if idx < 0 {
		panic("invalid index")
	} else if idx != 0 {
		arr := make([]byte, 8)
		binary.LittleEndian.PutUint64(arr, uint64(idx))
		a.SetBytes(arr)
	}
	return a
}

// toHash converts a key/value index to a hash
func toHash(idx int64) common.Hash {
	var h common.Hash
	if idx < 0 {
		panic("invalid index")
	} else if idx != 0 {
		// TODO: Improve encoding so that index conversion becomes sparse.
		arr := make([]byte, 32)
		binary.LittleEndian.PutUint64(arr, uint64(idx))
		h.SetBytes(arr)
	}
	return h
}

// delete account information when suicide was invoked
func (ss *stochasticState) deleteAccounts() {
	// remove account information when suicide was invoked in the block.
	for _, addrIdx := range ss.suicided {
		if err := ss.contracts.DeleteIndex(addrIdx); err != nil {
			panic("Failed deleting index")
		}
	}
	ss.suicided = []int64{}
}

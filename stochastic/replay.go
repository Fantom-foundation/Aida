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

// stochasticAccount keeps necessary account information for the simulation in memory
type stochasticAccount struct {
	balance int64 // current balance of account
}

// stochasticState keeps the execution state for the stochastic simulation
type stochasticState struct {
	db             state.StateDB                // StateDB database
	contracts      *generator.IndirectAccess    // index access generator for contracts
	keys           *generator.RandomAccess      // index access generator for keys
	values         *generator.RandomAccess      // index access generator for values
	snapshotLambda float64                      // lambda parameter for snapshot delta distribution
	txNum          uint32                       // current transaction number
	blockNum       uint64                       // current block number
	epochNum       uint64                       // current epoch number
	snapshot       []int                        // stack of active snapshots
	accounts       map[int64]*stochasticAccount // account information using address index as key
	balanceLog     map[int64][]int64            // balance log keeping track of balances for snapshots
	suicided       []int64                      // list of suicided accounts
	traceDebug     bool                         // trace-debug flag
	rg             *rand.Rand                   // random generator for sampling
}

// RunStochasticReplay runs the stochastic simulation for StateDB operations.
// It requires the simulation model and simulation length. The trace-debug flag
// enables/disables the printing of StateDB operations and their arguments on
// the screen.
func RunStochasticReplay(db state.StateDB, e *EstimationModelJSON, simLength int, cfg *utils.Config) error {
	// random generator
	rg := rand.New(rand.NewSource(cfg.RandomSeed))
	log.Printf("using random seed %d", cfg.RandomSeed)

	// retrieve operations and stochastic matrix from simulation object
	operations := e.Operations
	A := e.StochasticMatrix

	// produce random access generators for contract addresses,
	// storage-keys, and storage addresses.
	// (NB: Contracts need an indirect access wrapper because
	// contract addresses can be deleted by suicide.)
	contracts := generator.NewIndirectAccess(generator.NewRandomAccess(rg,
		e.Contracts.NumKeys,
		e.Contracts.Lambda,
		e.Contracts.QueueDistribution,
	))
	keys := generator.NewRandomAccess(rg,
		e.Keys.NumKeys,
		e.Keys.Lambda,
		e.Keys.QueueDistribution,
	)
	values := generator.NewRandomAccess(rg,
		e.Values.NumKeys,
		e.Values.Lambda,
		e.Values.QueueDistribution,
	)

	// setup state
	ss := NewStochasticState(rg, db, contracts, keys, values, e.SnapshotLambda, cfg.Debug)

	// create accounts in StateDB
	ss.prime()

	// set initial state to BeginEpoch
	state := initialState(operations, "BE")
	if state == -1 {
		return fmt.Errorf("BeginEpoch cannot be observed in stochastic matrix/recording failed.")
	}

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

	block := 0
	for {

		// decode opcode
		op, addrCl, keyCl, valueCl := DecodeOpcode(operations[state])

		// execute operation with its argument classes
		ss.execute(op, addrCl, keyCl, valueCl)

		// check for end of simulation
		if op == EndBlockID {
			block++
			if block >= simLength {
				break
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
	return runErr
}

// NewStochasticState creates a new state for execution StateDB operations
func NewStochasticState(rg *rand.Rand, db state.StateDB, contracts *generator.IndirectAccess, keys *generator.RandomAccess, values *generator.RandomAccess, snapshotLambda float64, traceDebug bool) stochasticState {

	// retrieve number of contracts
	n := contracts.NumElem()

	// initialise accounts in memory with balances greater than zero
	accounts := make(map[int64]*stochasticAccount, n+1)
	for i := int64(0); i <= n; i++ {
		accounts[i] = &stochasticAccount{
			balance: rand.Int63n(AddBalanceRange),
		}
	}

	// return stochastic state
	return stochasticState{
		db:             db,
		accounts:       accounts,
		contracts:      contracts,
		keys:           keys,
		values:         values,
		snapshotLambda: snapshotLambda,
		traceDebug:     traceDebug,
		balanceLog:     make(map[int64][]int64),
		suicided:       []int64{},
		blockNum:       1,
		epochNum:       1,
		rg:             rg,
	}
}

// prime StateDB accounts using account information
func (ss *stochasticState) prime() {
	log.Printf("Start priming...\n")
	db := ss.db
	db.BeginEpoch(0)
	db.BeginBlock(0)
	db.BeginTransaction(0)
	for addrIdx, detail := range ss.accounts {
		addr := toAddress(addrIdx)
		db.CreateAccount(addr)
		if detail.balance > 0 {
			db.AddBalance(addr, big.NewInt(detail.balance))
		}
	}
	db.Finalise(FinaliseFlag)
	db.EndTransaction()
	db.EndBlock()
	db.EndEpoch()
	log.Printf("End priming...\n")
}

// execute StateDB operations on a stochastic state.
func (ss *stochasticState) execute(op int, addrCl int, keyCl int, valueCl int) {
	var (
		addr  common.Address
		key   common.Hash
		value common.Hash
		db    state.StateDB = ss.db
	)

	// fetch indexes from index access generators
	addrIdx := ss.contracts.NextIndex(addrCl)
	keyIdx := ss.keys.NextIndex(keyCl)
	valueIdx := ss.values.NextIndex(valueCl)

	// convert index to address/hashes
	if addrCl != statistics.NoArgID {
		if addrCl == statistics.NewValueID {
			// create a new internal representation of an account
			// but don't create an account in StateDB; this is done
			// by CreateAccount.
			ss.accounts[addrIdx] = &stochasticAccount{
				balance: 0,
			}
		}
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
		value := rand.Int63n(AddBalanceRange)
		if ss.traceDebug {
			fmt.Printf(" value: %v", value)
		}
		ss.updateBalanceLog(addrIdx, value)
		db.AddBalance(addr, big.NewInt(value))

	case BeginBlockID:
		if ss.traceDebug {
			fmt.Printf(" id: %v", ss.blockNum)
		}
		db.BeginBlock(ss.blockNum)
		ss.txNum = 0
		ss.suicided = []int64{}

	case BeginEpochID:
		if ss.traceDebug {
			fmt.Printf(" id: %v", ss.epochNum)
		}
		db.BeginEpoch(ss.epochNum)

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

	case EndEpochID:
		db.EndEpoch()
		ss.epochNum++

	case EndTransactionID:
		db.EndTransaction()
		ss.txNum++
		ss.commitBalanceLog()
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
			snapshotIdx := snapshotNum - int(exponential.DiscreteSample(ss.rg, ss.snapshotLambda, int64(snapshotNum))) - 1
			snapshot := ss.snapshot[snapshotIdx]
			if ss.traceDebug {
				fmt.Printf(" id: %v", snapshot)
			}
			db.RevertToSnapshot(snapshot)

			// update active snapshots and perform a rollback in balance log
			ss.snapshot = ss.snapshot[0:snapshotIdx]
			ss.rollbackBalanceLog(snapshotIdx)
		}

	case SetCodeID:
		sz := rand.Intn(MaxCodeSize-1) + 1
		if ss.traceDebug {
			fmt.Printf(" code-size: %v", sz)
		}
		code := make([]byte, sz)
		_, err := rand.Read(code)
		if err != nil {
			log.Fatalf("error producing a random byte slice. Error: %v", err)
		}
		db.SetCode(addr, code)

	case SetNonceID:
		value := uint64(rand.Intn(SetNonceRange))
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
		balance := ss.getBalanceLog(addrIdx)
		if balance > 0 {
			// get a delta that does not exceed current balance
			// in the current snapshot
			value := rand.Int63n(balance)
			if ss.traceDebug {
				fmt.Printf(" value: %v", value)
			}
			db.SubBalance(addr, big.NewInt(value))
			ss.updateBalanceLog(addrIdx, -value)
		}

	case SuicideID:
		db.Suicide(addr)
		value := ss.getBalanceLog(addrIdx)
		ss.updateBalanceLog(addrIdx, -value)
		ss.suicided = append(ss.suicided, addrIdx)

	default:
		panic("invalid operation")
	}
	if ss.traceDebug {
		fmt.Println()
	}
}

// initialState returns the row/column index of the first state in the stochastic matrix.
func initialState(operations []string, opcode string) int {
	for i, opc := range operations {
		if opc == opcode {
			return i
		}
	}
	return -1
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
// TODO: Improve encoding so that index conversion becomes sparse.
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

// getBalanceLog computes the actual balance for the current snapshot
func (ss *stochasticState) getBalanceLog(addrIdx int64) int64 {
	balance := ss.accounts[addrIdx].balance
	for _, v := range ss.balanceLog[addrIdx] {
		balance += v
	}
	return balance
}

// updateBalanceLog adds a delta balance for an contract for the current snapshot.
func (ss *stochasticState) updateBalanceLog(addrIdx int64, delta int64) {
	snapshotNum := len(ss.snapshot) // retrieve number of active snapshots
	if snapshotNum > 0 {
		logLen := len(ss.balanceLog[addrIdx]) // retrieve number of log entries for addrIdx
		if logLen < snapshotNum {
			// fill log entry if too short with zeros
			ss.balanceLog[addrIdx] = append(ss.balanceLog[addrIdx], make([]int64, snapshotNum-logLen)...)
		} else if logLen != snapshotNum {
			panic("log wasn't rolled black")
		}
		// update delta of address for current snapshot
		ss.balanceLog[addrIdx][snapshotNum-1] += delta
	} else {
		// if no snapshot exists, just add delta to balance directly
		ss.accounts[addrIdx].balance += delta
	}
}

// commitBalanceLog updates the balances in the account and
// deletes the balance log.
func (ss *stochasticState) commitBalanceLog() {
	// update balances with balance log
	for idx, log := range ss.balanceLog {
		balance := ss.accounts[idx].balance
		for _, value := range log {
			balance += value
		}
		ss.accounts[idx].balance = balance
	}

	// destroy old log for next transaction
	ss.balanceLog = make(map[int64][]int64)
}

// rollbackBalanceLog rollbacks balance log to the k-th snapshot
func (ss *stochasticState) rollbackBalanceLog(k int) {
	// delete deltas of prior snapshots in balance log
	for idx, log := range ss.balanceLog {
		if len(log) > k {
			ss.balanceLog[idx] = ss.balanceLog[idx][0:k]
		}
	}
}

// delete account information when suicide was invoked
func (ss *stochasticState) deleteAccounts() {
	// remove account information when suicide was invoked in the block.
	for _, addrIdx := range ss.suicided {
		delete(ss.accounts, addrIdx)
		if err := ss.contracts.DeleteIndex(addrIdx); err != nil {
			panic("Failed deleting index")
		}
	}
	ss.suicided = []int64{}
}

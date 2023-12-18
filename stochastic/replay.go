package stochastic

import (
	"encoding/binary"
	"math/big"
	"math/rand"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/stochastic/exponential"
	"github.com/Fantom-foundation/Aida/stochastic/generator"
	"github.com/Fantom-foundation/Aida/stochastic/statistics"
	"github.com/ethereum/go-ethereum/common"
)

// Parameterisable simulation constants
var (
	BalanceRange int64 = 100000  // balance range for generating randomized values
	NonceRange   int   = 1000000 // nonce range for generating randomized nonces
)

// Simulation constants
const (
	MaxCodeSize  = 24576 // fixed upper limit by EIP-170
	FinaliseFlag = true  // flag for Finalise() StateDB operation
)

// State keeps the execution state for the stochastic simulation
type State struct {
	contracts      *generator.IndirectAccess // index access generator for contracts
	keys           *generator.RandomAccess   // index access generator for keys
	values         *generator.RandomAccess   // index access generator for values
	snapshotLambda float64                   // lambda parameter for snapshot delta distribution
	totalTx        uint64                    // total number of transactions
	syncPeriodNum  uint64                    // current sync-period number
	snapshot       []int                     // stack of active snapshots
	suicided       []int64                   // list of suicided accounts
	traceDebug     bool                      // trace-debug flag
	rg             *rand.Rand                // random generator for sampling
	log            logger.Logger
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

// CreateState creates a stochastic state and primes the StateDB
func CreateState(e *EstimationModelJSON, rg *rand.Rand, log logger.Logger) *State {
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
	ss := NewStochasticState(rg, contracts, keys, values, e.SnapshotLambda, log)

	// create accounts in StateDB
	ss.prime()

	return &ss
}

// GetStochasticMatrix returns the stochastic matrix with its operations and the initial state
func GetStochasticMatrix(e *EstimationModelJSON) ([]string, [][]float64, int) {
	operations := e.Operations
	A := e.StochasticMatrix
	// and set initial state to BeginSyncPeriod
	state := find(operations, OpMnemo(BeginSyncPeriodID))
	if state == -1 {
		panic("BeginSyncPeriod cannot be observed in stochastic matrix/recording failed.")
	}
	return operations, A, state
}

// NewStochasticState creates a new state for execution StateDB operations
func NewStochasticState(rg *rand.Rand, contracts *generator.IndirectAccess, keys *generator.RandomAccess, values *generator.RandomAccess, snapshotLambda float64, log logger.Logger) State {
	// return stochastic state
	return State{
		contracts:      contracts,
		keys:           keys,
		values:         values,
		snapshotLambda: snapshotLambda,
		traceDebug:     false,
		suicided:       []int64{},
		syncPeriodNum:  1,
		rg:             rg,
		log:            log,
	}
}

// prime StateDB accounts using account information
func (ss *State) prime() {
	// todo is priming done in the extension?
	//numInitialAccounts := ss.contracts.NumElem() + 1
	//ss.log.Notice("Start priming...")
	//ss.log.Noticef("\tinitializing %v accounts\n", numInitialAccounts)
	//pt := utils.NewProgressTracker(int(numInitialAccounts), ss.log)
	//db := ss.db
	//db.BeginSyncPeriod(0)
	//db.BeginBlock(0)
	//db.BeginTransaction(0)
	//
	//// initialise accounts in memory with balances greater than zero
	//for i := int64(0); i <= numInitialAccounts; i++ {
	//	addr := toAddress(i)
	//	db.CreateAccount(addr)
	//	db.AddBalance(addr, big.NewInt(ss.rg.Int63n(BalanceRange)))
	//	pt.PrintProgress()
	//}
	//ss.log.Notice("Finalizing...")
	//db.EndTransaction()
	//db.EndBlock()
	//db.EndSyncPeriod()
	//ss.log.Notice("End priming...")
}

// EnableDebug set traceDebug flag to true, and enable debug message when executing an operation
func (ss *State) EnableDebug() {
	ss.traceDebug = true
}

// Execute StateDB operations on a stochastic state.
func (ss *State) Execute(block, transaction int, data Data, db state.StateDB) {
	var (
		addr  common.Address
		key   common.Hash
		value common.Hash
		rg    = ss.rg
	)

	// fetch indexes from index access generators
	addrIdx := ss.contracts.NextIndex(data.Address)
	keyIdx := ss.keys.NextIndex(data.Key)
	valueIdx := ss.values.NextIndex(data.Value)

	// convert index to address/hashes
	if data.Address != statistics.NoArgID {
		addr = toAddress(addrIdx)
	}
	if data.Key != statistics.NoArgID {
		key = toHash(keyIdx)
	}
	if data.Value != statistics.NoArgID {
		value = toHash(valueIdx)
	}

	// print opcode and its arguments
	if ss.traceDebug {
		// print operation
		ss.log.Infof("opcode:%v (%v)", opText[data.Operation], EncodeOpcode(data.Operation, data.Address, data.Key, data.Value))

		// print indexes of contract address, storage key, and storage value.
		if data.Address != statistics.NoArgID {
			ss.log.Infof(" addr-idx: %v", addrIdx)
		}
		if data.Key != statistics.NoArgID {
			ss.log.Infof(" key-idx: %v", keyIdx)
		}
		if data.Value != statistics.NoArgID {
			ss.log.Infof(" value-idx: %v", valueIdx)
		}
	}

	switch data.Operation {
	case AddBalanceID:
		value := rg.Int63n(BalanceRange)
		if ss.traceDebug {
			ss.log.Infof("value: %v", value)
		}
		db.AddBalance(addr, big.NewInt(value))

	case BeginBlockID:
		if ss.traceDebug {
			ss.log.Infof(" id: %v", block)
		}
		db.BeginBlock(uint64(block))
		ss.suicided = []int64{}

	case BeginSyncPeriodID:
		if ss.traceDebug {
			ss.log.Infof(" id: %v", ss.syncPeriodNum)
		}
		db.BeginSyncPeriod(ss.syncPeriodNum)

	case BeginTransactionID:
		if ss.traceDebug {
			ss.log.Infof(" id: %v", transaction)
		}
		db.BeginTransaction(uint32(transaction))
		ss.snapshot = []int{}
		ss.suicided = []int64{}

	case CreateAccountID:
		db.CreateAccount(addr)

	case EmptyID:
		db.Empty(addr)

	case EndBlockID:
		db.EndBlock()
		ss.deleteAccounts()

	case EndSyncPeriodID:
		db.EndSyncPeriod()
		ss.syncPeriodNum++

	case EndTransactionID:
		db.EndTransaction()
		ss.totalTx++

	case ExistID:
		db.Exist(addr)

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
				ss.log.Infof(" id: %v", snapshot)
			}
			db.RevertToSnapshot(snapshot)

			// update active snapshots and perform a rollback in balance log
			ss.snapshot = ss.snapshot[0:snapshotIdx]
		}

	case SetCodeID:
		sz := rg.Intn(MaxCodeSize-1) + 1
		if ss.traceDebug {
			ss.log.Infof(" code-size: %v", sz)
		}
		code := make([]byte, sz)
		_, err := rg.Read(code)
		if err != nil {
			ss.log.Fatalf("error producing a random byte slice. Error: %v", err)
		}
		db.SetCode(addr, code)

	case SetNonceID:
		value := uint64(rg.Intn(NonceRange))
		db.SetNonce(addr, value)

	case SetStateID:
		db.SetState(addr, key, value)

	case SnapshotID:
		id := db.Snapshot()
		if ss.traceDebug {
			ss.log.Infof(" id: %v", id)
		}
		ss.snapshot = append(ss.snapshot, id)

	case SubBalanceID:
		shadowDB := db.GetShadowDB()
		var balance int64
		if shadowDB == nil {
			balance = db.GetBalance(addr).Int64()
		} else {
			balance = shadowDB.GetBalance(addr).Int64()
		}
		if balance > 0 {
			// get a delta that does not exceed current balance
			// in the current snapshot
			value := rg.Int63n(balance)
			if ss.traceDebug {
				ss.log.Infof(" value: %v", value)
			}
			db.SubBalance(addr, big.NewInt(value))
		}

	case SuicideID:
		db.Suicide(addr)
		if idx := find(ss.suicided, addrIdx); idx == -1 {
			ss.suicided = append(ss.suicided, addrIdx)
		}

	default:
		ss.log.Fatal("invalid operation")
	}
}

// NextState produces the next state in the Markovian process.
func NextState(rg *rand.Rand, A [][]float64, i int) int {
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
func (ss *State) deleteAccounts() {
	// remove account information when suicide was invoked in the block.
	for _, addrIdx := range ss.suicided {
		if err := ss.contracts.DeleteIndex(addrIdx); err != nil {
			ss.log.Fatal("failed deleting index")
		}
	}
	ss.suicided = []int64{}
}

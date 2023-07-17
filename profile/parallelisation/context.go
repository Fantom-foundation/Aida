package parallelisation

import (
	"errors"
	"time"

	"github.com/Fantom-foundation/Aida/profile/graphutil"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

// AddressSet is a set of contract/wallet addresses
type AddressSet map[common.Address]struct{}

// TxAddresses stores the used addresses of a transaction. The
// first transaction is stored in the first element and so on.
type TxAddresses []AddressSet

// TxTime stores time duration of transactions.
type TxTime []time.Duration

// Context stores the book-keeping information for parallelisation profiling
type Context struct {
	n              int                          // number of transactions
	txDependencies graphutil.StrictPartialOrder // transaction dependencies
	txAddresses    TxAddresses                  // contract/wallet addresses used in a transaction

	tSequential time.Duration // sequential transaction runtime of all transactions
	tOverheads  time.Duration // time overheads for  calculating dependencies

	tCompletion TxTime        // earliest completion time of a transaction
	tCritical   time.Duration // critical path runtime for transactions
}

var errBlockOverheadTime = errors.New("block or overhead time measurements went wrong")
var errBlockTxsTime = errors.New("block or txs time measurements went wrong")

// NewContext returns a new context.
func NewContext() *Context {
	return &Context{
		tCompletion:    TxTime{},
		txDependencies: graphutil.StrictPartialOrder{},
		txAddresses:    TxAddresses{},
	}
}

// interfere determines the interference between two transactions based on their address usage.
func interfere(u, v AddressSet) bool {
	// empty checks for address sets
	if len(u) == 0 || len(v) == 0 {
		return false
	}
	// range over the smaller address set
	if len(u) <= len(v) {
		// check whether an address exists that is contained in both sets
		for key := range u {
			if _, ok := v[key]; ok {
				return true
			}
		}
		return false
	} else {
		return interfere(v, u)
	}
}

// findTxAddresses gets wallet/contract addresses of a transaction
func findTxAddresses(tx *substate.Transaction) AddressSet {
	addresses := AddressSet{}
	for addr := range tx.Substate.InputAlloc {
		addresses[addr] = struct{}{}
	}
	for addr := range tx.Substate.OutputAlloc {
		addresses[addr] = struct{}{}
	}
	var zero common.Address
	if tx.Substate.Message.From != zero {
		addresses[tx.Substate.Message.From] = struct{}{}
	}
	if tx.Substate.Message.To != nil {
		addresses[*tx.Substate.Message.To] = struct{}{}
	}
	return addresses
}

// earliestTimeToRun computes the earliest time to run the current transaction
func (ctx *Context) earliestTimeToRun(addresses AddressSet) time.Duration {
	tEarliest := time.Duration(0)
	for i := 0; i < ctx.n; i++ {
		// check whether previous transaction interfere
		if interfere(addresses, ctx.txAddresses[i]) {
			// update earliest time to start a transaction
			if tEarliest < ctx.tCompletion[i] {
				tEarliest = ctx.tCompletion[i]
			}
		}

	}
	return tEarliest
}

// dependencies finds the transaction dependencies of the current transaction
func (ctx *Context) dependencies(addresses AddressSet) graphutil.OrdinalSet {
	dependentOn := graphutil.OrdinalSet{}
	for i := 0; i < ctx.n; i++ {
		// check whether previous transaction interfere
		if interfere(addresses, ctx.txAddresses[i]) {
			// remember direct and indirect transaction dependencies of a transaction
			dependentOn[i] = struct{}{}
			for j := range ctx.txDependencies[i] {
				dependentOn[j] = struct{}{}
			}
		}
	}
	return dependentOn
}

// RecordTransaction collects addresses and computes earliest time
func (ctx *Context) RecordTransaction(tx *substate.Transaction, tTransaction time.Duration) error {
	overheadTimer := time.Now()

	// update sequential time
	ctx.tSequential += tTransaction

	// retrieve contract/wallet addresses of transaction
	addresses := findTxAddresses(tx)

	// compute the earliest point in time to execute transaction
	tEarliest := ctx.earliestTimeToRun(addresses)

	// set earliest time to completion
	ctx.tCompletion = append(ctx.tCompletion, tEarliest+tTransaction)

	// update critical path if needed
	if ctx.tCritical < tEarliest+tTransaction {
		ctx.tCritical = tEarliest + tTransaction
	}

	// compute the dependencies of transaction, and
	// update transaction dependencies and addresses
	dependentOn := ctx.dependencies(addresses)
	ctx.txDependencies = append(ctx.txDependencies, dependentOn)
	ctx.txAddresses = append(ctx.txAddresses, addresses)

	// increment number of transactions
	ctx.n++

	// Measure runtime of ProcessTransaction and add it to tOverheads
	ctx.tOverheads += time.Since(overheadTimer)

	return nil
}

// ProfileData for one block
type ProfileData struct {
	curBlock    uint64  // current block number
	tBlock      int64   // block runtime
	tSequential int64   // total transaction runtime
	tCritical   int64   // critical path runtime for transactions
	tCommit     int64   // commit runtime
	speedup     float64 // speedup value for experiment
	ubNumProc   int64   // upper bound on the number of processors (i.e. width of task graph)
	numTx       int     // number of transactions
}

// GetProfileData produces a profile record for the SQLITE3 DB.
func (ctx *Context) GetProfileData(curBlock uint64, tBlock time.Duration) (*ProfileData, error) {

	// remove overheads from block runtime
	if tBlock < ctx.tOverheads {
		return nil, errBlockOverheadTime
	}
	tBlock -= ctx.tOverheads

	// time consistency check
	if tBlock < ctx.tSequential {
		return nil, errBlockTxsTime
	}

	// compute commit time
	tCommit := tBlock - ctx.tSequential

	// compute speedup
	speedup := float64(tBlock) / float64(tCommit+ctx.tCritical)

	// compute upper bound for number of processors using Dilworth's theorem:
	// the size of minimum chain cover is equal to the size of the maximum anti-chain
	// and the maximum anti-chain is the largest number of transactions that can
	// run independently.
	ubNumProc := int64(len(graphutil.MinChainCover(ctx.txDependencies)))

	// write data into SQLiteDB
	// profileData for parallel execution speedup experiment
	data := ProfileData{
		curBlock:    curBlock,
		tBlock:      tBlock.Nanoseconds(),
		tSequential: ctx.tSequential.Nanoseconds(),
		tCritical:   ctx.tCritical.Nanoseconds(),
		tCommit:     tCommit.Nanoseconds(),
		speedup:     speedup,
		ubNumProc:   ubNumProc,
		numTx:       ctx.n,
	}
	return &data, nil
}

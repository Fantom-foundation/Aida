package blockprofile

import (
	"errors"
	"time"

	"github.com/Fantom-foundation/Aida/profile/graphutil"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

// AddressSet is a set of contract/wallet addresses.
type AddressSet map[common.Address]struct{}

// TxAddresses stores the used addresses of a transaction. The
// first transaction is stored in the first element and so on.
type TxAddresses []AddressSet

// TxTime stores time duration of transactions.
type TxTime []time.Duration

// TxType stores type of transaction.
type TxType int

const (
	TransferTx    TxType = iota // a transaction which transafers balance
	CreateTx                    // a transaction which creates new contracts
	CallTx                      // a transaction which executes contracts
	MaintenanceTx               // an internal transaction which performs maintenance
)

// readable labels of transaction types.
var TypeLabel = map[TxType]string{
	TransferTx:    "transafer",
	CreateTx:      "create",
	CallTx:        "call",
	MaintenanceTx: "maintenance",
}

// Context stores the book-keeping information for block processing profiling.
type Context struct {
	n              int                          // number of transactions
	txDependencies graphutil.StrictPartialOrder // transaction dependencies
	txAddresses    TxAddresses                  // contract/wallet addresses used in a transaction

	tSequential   time.Duration   // sequential runtime of transactions
	tOverheads    time.Duration   // time overheads for profiling
	tCritical     time.Duration   // critical path runtime for transactions
	tCompletion   TxTime          // earliest completion time of a transaction
	tTransactions []time.Duration // runtime of a transaction
	tTypes        []TxType        // transaction type per transaction

	gasTransactions []uint64 // gas used for transactions
	gasBlock        uint64   // gas used for a block
}

var (
	errBlockOverheadTime = errors.New("block or overhead time measurements went wrong")
	errBlockTxsTime      = errors.New("block or txs time measurements went wrong")
	errInvalidLen        = errors.New("tTransactions or transactinGas length is invalid")
)

// NewContext returns a new context.
func NewContext() *Context {
	return &Context{
		tCompletion:     TxTime{},
		txDependencies:  graphutil.StrictPartialOrder{},
		txAddresses:     TxAddresses{},
		tTransactions:   []time.Duration{},
		tTypes:          []TxType{},
		gasTransactions: []uint64{},
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

// findTxAddresses gets wallet/contract addresses of a transaction.
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

// earliestTimeToRun computes the earliest time to run the current transaction.
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

// dependencies finds the transaction dependencies of the current transaction.
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

// RecordTransaction collects addresses and computes earliest time.
func (ctx *Context) RecordTransaction(tx *substate.Transaction, tTransaction time.Duration) error {
	overheadTimer := time.Now()

	// update time for block and transaction
	ctx.tSequential += tTransaction
	ctx.tTransactions = append(ctx.tTransactions, tTransaction)
	ctx.tTypes = append(ctx.tTypes, getTransactionType(tx))

	// update gas used for block and transaction
	gasUsed := tx.Substate.Result.GasUsed
	ctx.gasBlock += gasUsed
	ctx.gasTransactions = append(ctx.gasTransactions, gasUsed)

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

// ProfileData for a block.
type ProfileData struct {
	curBlock        uint64   // current block number
	numTx           int      // number of transactions
	tBlock          int64    // block runtime
	tSequential     int64    // total transaction runtime
	tCritical       int64    // critical path runtime for transactions
	tCommit         int64    // commit runtime
	tTransactions   []int64  // runtime per transaction
	tTypes          []TxType // a list of transaction type
	speedup         float64  // speedup value for experiment
	ubNumProc       int64    // upper bound on the number of processors (i.e. width of task graph)
	gasTransactions []uint64 // gas consumption per transaction
	gasBlock        uint64   // gas consumption of block
}

// GetProfileData produces a profile record for the profiling database.
func (ctx *Context) GetProfileData(curBlock uint64, tBlock time.Duration) (*ProfileData, error) {

	// perform consistency check
	if len(ctx.tTransactions) != len(ctx.gasTransactions) {
		return nil, errInvalidLen
	}
	if tBlock < ctx.tOverheads {
		return nil, errBlockOverheadTime
	}
	if tBlock < ctx.tSequential {
		return nil, errBlockTxsTime
	}

	// remove overheads from block runtime
	tBlock -= ctx.tOverheads

	// compute commit time
	// TODO: Includes BeginBlock()/BeginSyncPeriod() as well
	tCommit := tBlock - ctx.tSequential

	// compute speedup
	speedup := float64(tBlock) / float64(tCommit+ctx.tCritical)

	// compute upper bound for number of processors using Dilworth's theorem:
	// the size of minimum chain cover is equal to the size of the maximum anti-chain
	// and the maximum anti-chain is the largest number of transactions that can
	// run independently.
	ubNumProc := int64(len(graphutil.MinChainCover(ctx.txDependencies)))

	// transfer fields from context to profile record
	tTransactions := make([]int64, 0, len(ctx.tTransactions))
	gasTransactions := make([]uint64, 0, len(ctx.gasTransactions))
	for i, tTransaction := range ctx.tTransactions {
		tTransactions = append(tTransactions, tTransaction.Nanoseconds())
		gasTransactions = append(gasTransactions, ctx.gasTransactions[i])
	}
	data := ProfileData{
		curBlock:        curBlock,
		numTx:           ctx.n,
		tBlock:          tBlock.Nanoseconds(),
		tSequential:     ctx.tSequential.Nanoseconds(),
		tCritical:       ctx.tCritical.Nanoseconds(),
		tCommit:         tCommit.Nanoseconds(),
		tTransactions:   tTransactions,
		tTypes:          ctx.tTypes,
		speedup:         speedup,
		ubNumProc:       ubNumProc,
		gasTransactions: gasTransactions,
		gasBlock:        ctx.gasBlock,
	}
	return &data, nil
}

// getTransactionType reads a message and determines a transaction type.
func getTransactionType(tx *substate.Transaction) TxType {
	msg := tx.Substate.Message
	to := msg.To
	from := msg.From
	alloc := tx.Substate.InputAlloc

	zero := common.HexToAddress("0x0000000000000000000000000000000000000000")

	if to != nil {
		account, exist := alloc[*to]
		// regular transaction
		if !exist || len(account.Code) == 0 {
			return TransferTx
			// CALL transaction with contract bytecode
		} else {
			// a maintenance transaction is sent from address zero
			if from == zero {
				return MaintenanceTx
			}
			return CallTx
		}
	}
	// CREATE transaction
	return CreateTx
}

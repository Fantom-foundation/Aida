package executor

//go:generate mockgen -source substate_provider.go -destination substate_provider_mocks.go -package executor

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// ----------------------------------------------------------------------------
//                               Interface
// ----------------------------------------------------------------------------

// ActionProvider is an interface for components capable of enumerating
// provider-data for ranges of transactions.
type ActionProvider interface {
	// Run iterates through transaction in the block range [from,to) in order
	// and forwards provider information for each transaction in the range to
	// the provided consumer. Execution aborts if the consumer returns an error
	// or an error during the provider retrieval process occurred.
	Run(from int, to int, consumer Consumer) error
	// Close releases resources held by the Substate implementation. After this
	// no more operations are allowed on the same instance.
	Close()
}

// Consumer is a type alias for the type of function to which provider information
// can be forwarded by the ActionProvider.
type Consumer func(TransactionInfo, operation.Operation) error

// TransactionInfo summarizes the per-transaction information provided by a
// ActionProvider.
type TransactionInfo struct {
	Block       int
	Transaction int
	Substate    *substate.Substate
}

// ----------------------------------------------------------------------------
//                              Implementation
// ----------------------------------------------------------------------------

// OpenSubstateDb opens a provider database as configured in the given parameters.
func OpenSubstateDb(config *utils.Config, ctxt *cli.Context) (res ActionProvider, err error) {
	// Substate is panicking if we are opening a non-existing directory. To mitigate
	// the damage, we recover here and forward an error instead.
	defer func() {
		if issue := recover(); issue != nil {
			res = nil
			err = fmt.Errorf("failed to open provider DB: %v", issue)
		}
	}()
	substate.SetSubstateDb(config.AidaDb)
	substate.OpenSubstateDBReadOnly()
	return &substateProvider{ctxt, config.Workers}, nil
}

// substateProvider is an adapter of Aida's ActionProvider interface defined above to the
// current provider implementation offered by github.com/Fantom-foundation/Substate.
type substateProvider struct {
	ctxt                *cli.Context
	numParallelDecoders int
}

func (s substateProvider) Run(from int, to int, consumer Consumer) error {
	iter := substate.NewSubstateIterator(uint64(from), s.numParallelDecoders)
	defer iter.Release()
	for iter.Next() {
		tx := iter.Value()
		if tx.Block >= uint64(to) {
			return nil
		}
		if err := consumer(TransactionInfo{int(tx.Block), int(tx.Transaction), tx.Substate}, nil); err != nil {
			return err
		}
	}
	return nil
}

func (substateProvider) Close() {
	substate.CloseSubstateDB()
}

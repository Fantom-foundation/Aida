package executor

//go:generate mockgen -source substate_provider.go -destination substate_provider_mocks.go -package executor

import substate "github.com/Fantom-foundation/Substate"

// SubstateProvider is an interface for components capable of enumerating
// substate-data for ranges of transactions.
type SubstateProvider interface {
	// Run iterates through transaction in the block range [from,to) in order
	// and forwards substate information for each transaction in the range to
	// the provided consumer. Execution aborts if the consumer returns an error
	// or an error during the substate retrieval process occured.
	Run(from int, to int, comsumer Consumer) error
}

// Consumer is a type alias for the type of function to which substate information
// can be forwarded by the SubstateProvider.
type Consumer func(block int, transaction int, substate *substate.Substate) error

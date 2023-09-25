package action_provider

import "github.com/Fantom-foundation/Aida/tracer/operation"

// ActionProvider is an interface for components capable of enumerating
// provider-data for ranges of transactions.
type ActionProvider interface {
	// Run iterates through action in the block range [from,to) in order
	// and forwards provider information for each transaction in the range to
	// the provided consumer. Execution aborts if the consumer returns an error
	// or an error during the provider retrieval process occurred.
	Run(from int, to int, consumer Consumer) error
	// Close releases all held resources. After this
	// no more operations are allowed on the same instance.
	Close()
}

// Consumer is a type alias for the type of function to which provider information
// can be forwarded by the ActionProvider.
type Consumer func(TransactionInfo, operation.Operation) error

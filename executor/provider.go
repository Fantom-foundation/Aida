package executor

//go:generate mockgen -source provider.go -destination provider_mocks.go -package executor

type Provider[T any] interface {
	// Run iterates through transactionResult in the block range [from,to) in order
	// and forwards payload information for each transactionResult in the range to
	// the provided consumer. Execution aborts if the consumer returns an error
	// or an error during the payload retrieval process occurred.
	Run(from int, to int, consumer Consumer[T]) error
	// Close releases resources held by the provider implementation. After this
	// no more operations are allowed on the same instance.
	Close()
}

// Consumer is a type alias for the type of function to which payload information
// can be forwarded by a Provider.
type Consumer[T any] func(TransactionInfo[T]) error

// TransactionInfo summarizes the per-transactionResult information provided by a
// Provider.
type TransactionInfo[T any] struct {
	Block       int
	Transaction int
	Data        T
}

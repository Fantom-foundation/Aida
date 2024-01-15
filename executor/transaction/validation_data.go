package transaction

// InputValidationData represents what is necessary to implement if input validation is required.
type InputValidationData interface {
	// GetInputAlloc returns the state of the Alloc BEFORE executing the transaction.
	// This is mainly used for confirming that StateDb has correct data before execution.
	// And/Or for creating an InMemory StateDb which lifespan is a single transaction.
	GetInputAlloc() Alloc
}

// OutputValidationData represents what is necessary to implement if output validation is required.
type OutputValidationData interface {
	// GetOutputAlloc returns the state of the Alloc AFTER executing the transaction.
	// This is mainly used for confirming that StateDb has correct data AFTER execution.
	GetOutputAlloc() Alloc

	// GetResult returns the Result of the transaction.
	// This is used for comparing result returned by the StateDb.
	GetResult() Result
}

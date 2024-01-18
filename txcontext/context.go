package txcontext

import "github.com/ethereum/go-ethereum/core/types"

// WithValidation implements all three interfaces necessary for
// Input/Output validation and Transaction execution
type WithValidation interface {
	InputValidationData
	ExecutionData
	OutputValidationData
}

// InputValidationData represents what is necessary to implement if input validation is required.
type InputValidationData interface {
	// GetInputState returns the state of the WorldState BEFORE executing the transaction.
	// This is mainly used for confirming that StateDb has correct data before execution.
	// And/Or for creating an InMemory StateDb which lifespan is a single transaction.
	GetInputState() WorldState
}

// ExecutionData represents what is necessary to implement to be able to execute a transaction using the Executor.
type ExecutionData interface {
	// GetBlockEnvironment returns the transaction environment.
	// This is used for creating the correct block environment for execution.
	GetBlockEnvironment() BlockEnvironment

	// GetMessage returns the message of the transaction.
	// Message holds data needed by the EVM to execute the transaction.
	GetMessage() types.Message
}

// OutputValidationData represents what is necessary to implement if output validation is required.
type OutputValidationData interface {
	// GetOutputState returns the state of the WorldState AFTER executing the transaction.
	// This is mainly used for confirming that StateDb has correct data AFTER execution.
	GetOutputState() WorldState

	// GetReceipt returns the Receipt of the transaction.
	// This is used for comparing result returned by the StateDb.
	GetReceipt() Receipt
}

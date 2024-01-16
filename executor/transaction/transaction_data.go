package transaction

import (
	"github.com/ethereum/go-ethereum/core/types"
)

// ExecutionData represents what is necessary to implement to be able to execute a transaction using the Executor.
type ExecutionData interface {
	// GetEnv returns the transaction environment.
	// This is used for creating the correct block environment for execution.
	GetEnv() BlockEnvironment

	// GetMessage returns the message of the transaction.
	// Message holds data needed by the EVM to execute the transaction.
	GetMessage() types.Message
}

package executor

import (
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/types"
)

// TransactionData represents the structure of a single transaction executed by the Executor.
type TransactionData interface {
	// GetInputAlloc returns the state of an Alloc BEFORE executing the transaction.
	// This is mainly used for confirming that StateDb has correct data before execution.
	// And/Or for creating an InMemory StateDb which lifespan is a single transaction.
	GetInputAlloc() substate.SubstateAlloc

	// GetOutputAlloc returns the state of an Alloc AFTER executing the transaction.
	// This is mainly used for confirming that StateDb has correct data AFTER execution.
	GetOutputAlloc() substate.SubstateAlloc

	// GetEnv returns the transaction environment.
	// This is used for creating the correct block environment for execution.
	GetEnv() *substate.SubstateEnv

	// GetMessage returns the message of the transaction.
	// Message holds data needed by the EVM to execute the transaction.
	GetMessage() types.Message

	// GetResult returns the result of the transaction.
	// This is used for comparing result returned by the StateDb.
	GetResult() *substate.SubstateResult
}

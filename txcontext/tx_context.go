package txcontext

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

// TxContext implements all three interfaces necessary for
// Input/Output validation and Transaction execution
type TxContext interface {
	InputState
	Transaction
	OutputState
}

// InputState represents what is necessary to implement if input validation is required.
type InputState interface {
	// GetInputState returns the state of the WorldState BEFORE executing the transaction.
	// This is mainly used for confirming that StateDb has correct data before execution.
	// And/Or for creating an InMemory StateDb which lifespan is a single transaction.
	GetInputState() WorldState
}

// Transaction represents what is necessary to implement to be able to execute a transaction using the Executor.
type Transaction interface {
	// GetBlockEnvironment returns the transaction environment.
	// This is used for creating the correct block environment for execution.
	GetBlockEnvironment() BlockEnvironment

	// GetMessage returns the message of the transaction.
	// Message holds data needed by the EVM to execute the transaction.
	GetMessage() core.Message

	// GetOutputState returns the state of the WorldState AFTER executing the transaction.
	// This is mainly used for confirming that StateDb has correct data AFTER execution
	// and executing pseudo transaction in the beginning of the chain.
	// Note: If no pseudo transactions (transactions marked as number 99) are present
	// within the data-set and PostTx validation is not planned this can return nil.
	GetOutputState() WorldState
}

// OutputState represents what is necessary to implement if output validation is required.
type OutputState interface {
	// GetResult returns the Result of the execution.
	// This is used for comparing result returned by the StateDb.
	GetResult() Result

	// GetStateHash returns expected State Hash. This is only used
	// by Eth JSON tests and can be ignored for most implementations.
	GetStateHash() common.Hash
}

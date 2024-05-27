// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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

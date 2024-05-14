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

package executor

import (
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/txcontext"
)

//go:generate mockgen -source test_consumer.go -destination test_consumer_mocks.go -package executor

//---------------------------------------------------------------------------------//
// This file serves for creating a mock Consumer with specific type. Every possible
// type of Consumer should be included therefore should be tested independently.
//---------------------------------------------------------------------------------//

type TxConsumer interface {
	Consume(block int, transaction int, substate txcontext.TxContext) error
}

func toSubstateConsumer(c TxConsumer) Consumer[txcontext.TxContext] {
	return func(info TransactionInfo[txcontext.TxContext]) error {
		return c.Consume(info.Block, info.Transaction, info.Data)
	}
}

type RPCReqConsumer interface {
	Consume(block int, transaction int, req *rpc.RequestAndResults) error
}

func toRPCConsumer(c RPCReqConsumer) Consumer[*rpc.RequestAndResults] {
	return func(info TransactionInfo[*rpc.RequestAndResults]) error {
		return c.Consume(info.Block, info.Transaction, info.Data)
	}
}

type OperationConsumer interface {
	Consume(block int, transaction int, operations []operation.Operation) error
}

func toOperationConsumer(c OperationConsumer) Consumer[[]operation.Operation] {
	return func(info TransactionInfo[[]operation.Operation]) error {
		return c.Consume(info.Block, info.Transaction, info.Data)
	}
}

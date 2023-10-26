package executor

import (
	"github.com/Fantom-foundation/Aida/rpc_iterator"
	substate "github.com/Fantom-foundation/Substate"
)

//go:generate mockgen -source test_consumer.go -destination test_consumer_mocks.go -package executor

//---------------------------------------------------------------------------------//
// This file serves for creating a mock Consumer with specific type. Every possible
// type of Consumer should be included therefore should be tested independently.
//---------------------------------------------------------------------------------//

type TxConsumer interface {
	Consume(block int, transaction int, substate *substate.Substate) error
}

func toSubstateConsumer(c TxConsumer) Consumer[*substate.Substate] {
	return func(info TransactionInfo[*substate.Substate]) error {
		return c.Consume(info.Block, info.Transaction, info.Data)
	}
}

type RPCReqConsumer interface {
	Consume(block int, transaction int, req *rpc_iterator.RequestWithResponse) error
}

func toRPCConsumer(c RPCReqConsumer) Consumer[*rpc_iterator.RequestWithResponse] {
	return func(info TransactionInfo[*rpc_iterator.RequestWithResponse]) error {
		return c.Consume(info.Block, info.Transaction, info.Data)
	}
}

package executor

import (
	"github.com/Fantom-foundation/Aida/tracer/operation"
)

//go:generate mockgen -source test_consumer.go -destination test_consumer_mocks.go -package executor

type OperationConsumer interface {
	Consume(block int, transaction int, operations []operation.Operation) error
}

func toOperationConsumer(c OperationConsumer) Consumer[[]operation.Operation] {
	return func(info TransactionInfo[[]operation.Operation]) error {
		return c.Consume(info.Block, info.Transaction, info.Data)
	}
}

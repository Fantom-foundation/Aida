package executor

//go:generate mockgen -source operation_provider.go -destination operation_provider_mocks.go -package executor

import (
	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
)

type operationProvider struct {
	traceFiles []string
}

// Run unifies operation by transactions. If operation.BeginTransactionID
// appears current slice of operations is sent to the consumer
func (p operationProvider) Run(from int, _ int, consumer Consumer[[]operation.Operation]) error {
	iter := tracer.NewTraceIterator(p.traceFiles, uint64(from))
	defer iter.Release()

	tx := make([]operation.Operation, 0)
	currentBlockNumber := from
	for iter.Next() {
		op := iter.Value()

		// we need to know the block number
		if bb, ok := op.(*operation.BeginBlock); ok {
			currentBlockNumber = int(bb.BlockNumber)
		}

		// new tx appeared - send current operation array to consumer
		if bt, ok := op.(*operation.BeginTransaction); ok {
			if err := consumer(TransactionInfo[[]operation.Operation]{int(bt.TransactionNumber), currentBlockNumber, tx}); err != nil {
				return err
			}
			tx = make([]operation.Operation, 0)
		}

		tx = append(tx, op)

	}

	return nil
}

func (p operationProvider) Close() {
	// ignored
}

func OpenOperations(config *utils.Config) (Provider[[]operation.Operation], error) {
	traceFiles, err := tracer.GetTraceFiles(config)
	if err != nil {
		return operationProvider{}, err
	}

	return &operationProvider{
		traceFiles: traceFiles,
	}, nil
}

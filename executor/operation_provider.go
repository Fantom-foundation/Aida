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
func (p operationProvider) Run(from int, to int, consumer Consumer[[]operation.Operation]) error {
	iter := tracer.NewTraceIterator(p.traceFiles, uint64(from))
	defer iter.Release()

	tx := make([]operation.Operation, 0)
	currentBlockNumber := from
	var (
		transactionNumber int
		lastOperation     bool
	)
	for iter.Next() {
		op := iter.Value()

		// if next operation after operation.EndTransaction is operation.EndBlock append as well
		if lastOperation {
			var ok bool

			// append operation.EndBlock as well
			if _, ok = op.(*operation.EndBlock); ok {
				tx = append(tx, op)
			}

			if err := consumer(TransactionInfo[[]operation.Operation]{currentBlockNumber, transactionNumber, tx}); err != nil {
				return err
			}

			// this condition must be kept for replay_substate tool;
			// it indicates that we require only one txcontext to be passed to consumer
			if from == to {
				return nil
			}

			tx = make([]operation.Operation, 0)
			lastOperation = false

			// operation has been already appended, skip the rest of the loop
			if ok {
				continue
			}
		}

		switch t := op.(type) {
		case *operation.BeginTransaction:
			// extract txcontext number with operation.BeginTransaction
			transactionNumber = int(t.TransactionNumber)
		case *operation.EndTransaction:
			// send the united operations to consumer with operation.EndTransaction
			lastOperation = true
		case *operation.BeginBlock:
			// extract block number with operation.BeginBlock
			currentBlockNumber = int(t.BlockNumber)
			if currentBlockNumber > to {
				return nil
			}
		default:
		}

		tx = append(tx, op)
	}

	return nil
}

func (p operationProvider) Close() {
	// ignored
}

func OpenOperations(cfg *utils.Config) (Provider[[]operation.Operation], error) {
	traceFiles, err := tracer.GetTraceFiles(cfg)
	if err != nil {
		return operationProvider{}, err
	}

	return &operationProvider{
		traceFiles: traceFiles,
	}, nil
}

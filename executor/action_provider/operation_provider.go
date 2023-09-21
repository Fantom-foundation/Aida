package action_provider

//go:generate mockgen -source operation_provider.go -destination operation_provider_mocks.go -package action_provider

import (
	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/utils"
)

// OperationProvider is an interface for components
// capable of enumerating StateDb operations.
type OperationProvider interface {
	// Run iterates through operations in the block range [from,to) in order
	// and forwards provider information for each operation in the range to
	// the provided consumer. Execution aborts if the consumer returns an error
	// or an error during the provider retrieval process occurred.
	Run(from int, to int, consumer Consumer) error

	Close()
}

type operationProvider struct {
	traceFiles []string
	debug      bool
	debugFrom  int
}

func (p operationProvider) Run(from int, _ int, consumer Consumer) error {
	iter := tracer.NewTraceIterator(p.traceFiles, uint64(from))
	defer iter.Release()

	for iter.Next() {
		op := iter.Value()
		if err := consumer(TransactionInfo{}, op); err != nil {
			return err
		}
	}

	return nil
}

func (p operationProvider) Close() {
	// ignored
}

func OpenOperations(config *utils.Config) (ActionProvider, error) {
	traceFiles, err := tracer.GetTraceFiles(config)
	if err != nil {
		return substateProvider{}, err
	}

	return &operationProvider{
		traceFiles: traceFiles,
		debug:      config.Debug,
		debugFrom:  int(config.DebugFrom),
	}, nil
}

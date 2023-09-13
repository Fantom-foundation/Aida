package executor

import (
	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/utils"
)

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

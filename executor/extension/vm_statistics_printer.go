package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeVirtualMachineStatisticsPrinter creates an extension that prints VM specific
// profiling data at the end of a run, if this is supported by the VM implementation.
func MakeVirtualMachineStatisticsPrinter[T any](config *utils.Config) executor.Extension[T] {
	return &vmStatPrinter[T]{config: config}
}

type vmStatPrinter[T any] struct {
	NilExtension[T]
	config *utils.Config
}

func (p *vmStatPrinter[T]) PostRun(executor.State[T], *executor.Context, error) error {
	utils.PrintEvmStatistics(p.config)
	return nil
}

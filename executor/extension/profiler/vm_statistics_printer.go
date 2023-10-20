package profiler

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeVirtualMachineStatisticsPrinter creates an extension that prints VM specific
// profiling data at the end of a run, if this is supported by the VM implementation.
func MakeVirtualMachineStatisticsPrinter[T any](cfg *utils.Config) executor.Extension[T] {
	return &vmStatPrinter[T]{cfg: cfg}
}

type vmStatPrinter[T any] struct {
	extension.NilExtension[T]
	cfg *utils.Config
}

func (p *vmStatPrinter[T]) PostRun(executor.State[T], *executor.Context, error) error {
	utils.PrintEvmStatistics(p.cfg)
	return nil
}

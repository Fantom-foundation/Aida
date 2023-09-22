package profiler_extensions

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeVirtualMachineStatisticsPrinter creates an extension that prints VM specific
// profiling data at the end of a run, if this is supported by the VM implementation.
func MakeVirtualMachineStatisticsPrinter(config *utils.Config) executor.Extension {
	return &vmStatPrinter{config: config}
}

type vmStatPrinter struct {
	extension.NilExtension
	config *utils.Config
}

func (p *vmStatPrinter) PostRun(executor.State, *executor.Context, error) error {
	utils.PrintEvmStatistics(p.config)
	return nil
}

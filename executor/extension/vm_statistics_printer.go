package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeVirtualMachineStatisticsPrinter creates an extension that prints VM specific
// profiling data at the end of a run, if this is supported by the VM implementation.
func MakeVirtualMachineStatisticsPrinter(config *utils.Config) executor.Extension {
	return &vmStatPrinter{config: config}
}

type vmStatPrinter struct {
	NilExtension
	config *utils.Config
}

func (p *vmStatPrinter) PostRun(executor.State, error) error {
	utils.PrintEvmStatistics(p.config)
	return nil
}

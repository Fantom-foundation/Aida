package profiler

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"go.uber.org/mock/gomock"
)

func TestVirtualMachineStatisticsPrinter_WorksWithDefaultSetup(t *testing.T) {
	cfg := utils.Config{}
	ext := MakeVirtualMachineStatisticsPrinter[any](&cfg)
	ext.PostRun(executor.State[any]{}, nil, nil)
}

func TestVirtualMachineStatisticsPrinter_TriggersStatPrintingAtEndOfRun(t *testing.T) {
	ctrl := gomock.NewController(t)
	interpreter := vm.NewMockProfilingInterpreter(ctrl)
	vm.RegisterInterpreter("test-vm", interpreter)

	interpreter.EXPECT().DumpProfile()

	cfg := utils.Config{}
	cfg.VmImpl = "test-vm"
	ext := MakeVirtualMachineStatisticsPrinter[any](&cfg)

	ext.PostRun(executor.State[any]{}, nil, nil)
}

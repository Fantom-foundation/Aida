package extension

//go:generate mockgen -source vm_statistics_printer_test.go -destination vm_statistics_printer_mocks_test.go -package extension

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Tosca/go/vm/registry"
	"go.uber.org/mock/gomock"
)

func TestVirtualMachineStatisticsPrinter_WorksWithDefaultSetup(t *testing.T) {
	config := utils.Config{}
	ext := MakeVirtualMachineStatisticsPrinter(&config)
	ext.PostRun(executor.State{}, nil)
}

func TestVirtualMachineStatisticsPrinter_TriggersStatPrintingAtEndOfRun(t *testing.T) {
	ctrl := gomock.NewController(t)
	vm := NewMockProfilingVm(ctrl)
	registry.RegisterVirtualMachine("test-vm", vm)

	vm.EXPECT().DumpProfile()

	config := utils.Config{}
	config.VmImpl = "test-vm"
	ext := MakeVirtualMachineStatisticsPrinter(&config)

	ext.PostRun(executor.State{}, nil)
}

type ProfilingVm interface {
	registry.VirtualMachine
	registry.ProfilingVM
}
